package oai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// OpenAi is an HTTP client for the OpenAI-compatible provider API.
type OpenAi struct {
	httpClient *http.Client
}

// New creates a provider API client with a ten-second request timeout.
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
