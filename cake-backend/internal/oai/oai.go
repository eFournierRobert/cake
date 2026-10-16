package oai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

// OpenAi is an HTTP client for OpenAI-compatible model-catalog and Responses
// APIs, including streaming Responses API events.
type OpenAi struct {
	httpClient *http.Client
}

// New creates a provider API client using Go's default HTTP transport. It has
// no client-wide timeout; callers control request lifetime through ctx.
func New() *OpenAi {
	return &OpenAi{
		httpClient: &http.Client{},
	}
}

// GetModels fetches and decodes the model catalog at baseUrl + "/models".
// When apiKey is non-empty it is sent as a bearer token. Non-2xx responses,
// request failures, and invalid response JSON are returned as errors.
func (oai *OpenAi) GetModels(ctx context.Context, apiKey, baseUrl string) (GetModelsResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseUrl+"/models", nil)
	if err != nil {
		return GetModelsResponse{}, err
	}

	if len(apiKey) > 0 {
		req.Header.Set("Authorization", "Bearer "+string(apiKey))
	}

	resp, err := oai.httpClient.Do(req)

	if err != nil {
		return GetModelsResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return GetModelsResponse{}, fmt.Errorf("error: received %d from API %s", resp.StatusCode, baseUrl)
	}

	var getModelsResponse GetModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&getModelsResponse); err != nil {
		return GetModelsResponse{}, err
	}

	return getModelsResponse, nil
}

// CreateResponse sends a non-streaming Responses API request and decodes its
// JSON response. A non-success HTTP status, request failure, body-read failure,
// or invalid response JSON is returned as an error.
func (oai *OpenAi) CreateResponse(
	ctx context.Context,
	apiKey,
	baseUrl,
	model,
	reasoningEffort,
	instructions string,
	convoInput []ResponseInput) (ResponseResponse, error) {
	req, err := oai.createResponseRequest(ctx, apiKey, baseUrl, model, reasoningEffort, instructions, convoInput, false)
	if err != nil {
		return ResponseResponse{}, err
	}

	resp, err := oai.httpClient.Do(req)
	if err != nil {
		return ResponseResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logResponseBodyOnError(resp, resp.StatusCode)
		resp.Body.Close()
		return ResponseResponse{}, fmt.Errorf("error: received %d from API %s", resp.StatusCode, baseUrl)
	}

	fullBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ResponseResponse{}, err
	}

	var response ResponseResponse
	if err := json.Unmarshal(fullBody, &response); err != nil {
		return ResponseResponse{}, err
	}

	return response, nil
}

// CreateStreamingResponse posts a streaming Responses API request and returns
// a channel of recognized response events. The request is opened before this
// method returns; a goroutine then scans the provider's SSE body.
// response.output_text.delta, response.completed, response.failed, and
// response.incomplete events are forwarded, as are parsing/read errors via the
// event's Error field. Unrecognized event types and [DONE] are not forwarded.
// Cancelling ctx stops the request and stream reader.
func (oai *OpenAi) CreateStreamingResponse(
	ctx context.Context,
	apiKey,
	baseUrl,
	model,
	reasoningEffort,
	instructions string,
	convoInput []ResponseInput) (<-chan ResponseEvent, error) {
	req, err := oai.createResponseRequest(ctx, apiKey, baseUrl, model, reasoningEffort, instructions, convoInput, true)
	if err != nil {
		return nil, err
	}

	resp, err := oai.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logResponseBodyOnError(resp, resp.StatusCode)
		resp.Body.Close()
		return nil, fmt.Errorf("error: received %d from API %s", resp.StatusCode, baseUrl)
	}

	outputChannel := make(chan ResponseEvent)

	go func() {
		defer resp.Body.Close()
		defer close(outputChannel)

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 4096), 1024*1024)

		sendError := func(err error) {
			select {
			case outputChannel <- ResponseEvent{Error: err}:
			case <-ctx.Done():
			}
		}

		var dataLines []string

		for scanner.Scan() {
			line := scanner.Text()

			if line == "" {
				if len(dataLines) == 0 {
					continue
				}

				data := strings.Join(dataLines, "\n")
				dataLines = nil

				if data == "[DONE]" {
					return
				}

				var event ResponseEventData
				if err := json.Unmarshal([]byte(data), &event); err != nil {
					sendError(fmt.Errorf("unmarshal SSE data: %w", err))
					return
				}

				switch event.Type {
				case "response.output_text.delta":
					select {
					case outputChannel <- ResponseEvent{Data: event}:
					case <-ctx.Done():
						return
					}

				case "response.completed":
					select {
					case outputChannel <- ResponseEvent{Data: event}:
					case <-ctx.Done():
					}
					return

				case "response.failed", "response.incomplete":
					select {
					case outputChannel <- ResponseEvent{Data: event}:
					case <-ctx.Done():
					}
					return
				}

				continue
			}

			if strings.HasPrefix(line, ":") {
				continue
			}

			field, value, ok := strings.Cut(line, ":")
			if !ok {
				field, value = line, ""
			} else {
				value = strings.TrimPrefix(value, " ")
			}

			if field == "data" {
				dataLines = append(dataLines, value)
			}
		}

		if err := scanner.Err(); err != nil {
			sendError(fmt.Errorf("read SSE stream: %w", err))
		}
	}()

	return outputChannel, nil
}

// logResponseBodyOnError logs the response body when a provider rejects a
// request.
func logResponseBodyOnError(resp *http.Response, statusCode int) {
	body := resp.Body

	requestBody, err := io.ReadAll(body)
	if err != nil {
		log.Printf("OpenAI API request failed with status %d; could not read response body for logging: %v", statusCode, err)
		return
	}

	log.Printf("OpenAI API request failed with status %d; response body: %s", statusCode, requestBody)
}

// createResponseRequest builds a Responses API request. When streaming is
// true, the provider is asked to return SSE; a non-empty reasoningEffort is
// included in the request's reasoning settings.
func (oai *OpenAi) createResponseRequest(
	ctx context.Context,
	apiKey,
	baseUrl,
	model,
	reasoningEffort,
	instructions string,
	convoInput []ResponseInput,
	streaming bool) (*http.Request, error) {
	responseRequestBody := ResponseRequestBody{
		Model:           model,
		Instructions:    instructions,
		Stream:          streaming,
		Store:           false,
		MaxOutputTokens: 8192,
		Input:           convoInput,
		Reasoning:       nil,
	}

	if len(reasoningEffort) > 0 {
		responseRequestBody.Reasoning = &ResponseReasoning{Effort: reasoningEffort}
	}

	marshaledBody, err := json.Marshal(responseRequestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		baseUrl+"/responses",
		bytes.NewBuffer(marshaledBody))
	if err != nil {
		return nil, err
	}

	if len(apiKey) > 0 {
		req.Header.Set("Authorization", "Bearer "+string(apiKey))
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	return req, nil
}
