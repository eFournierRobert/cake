// Package oai provides the HTTP client and wire types for OpenAI-compatible
// model-catalog and Responses APIs, including streamed responses.
package oai

// GetModelsResponse is the response body returned by an OpenAI-compatible
// GET /models endpoint.
type GetModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

// Model describes a model entry returned by a provider's GET /models endpoint.
type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// ResponseEvent is one decoded provider event or a stream-reading error.
// Data is populated for recognized Responses API events; Error is populated
// when reading or decoding the SSE stream fails.
type ResponseEvent struct {
	Data  ResponseEventData
	Error error
}

// ResponseResponse contains the subset of a non-streaming Responses API
// response used by the application.
type ResponseResponse struct {
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
	Output []struct {
		Type    string `json:"type"`
		Role    string `json:"role"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
}

// ResponseEventData contains the fields consumed from a Responses API SSE
// event. Other provider event fields are ignored by this client.
type ResponseEventData struct {
	Type  string `json:"type"`  // Provider event type, such as response.output_text.delta.
	Delta string `json:"delta"` // Text fragment for response.output_text.delta events.
}

// ResponseRequestBody is the JSON request sent to the provider's
// POST /responses endpoint.
type ResponseRequestBody struct {
	Model           string             `json:"model"`
	Instructions    string             `json:"instructions"`
	Stream          bool               `json:"stream"`
	Store           bool               `json:"store"`
	MaxOutputTokens int                `json:"max_output_tokens"`
	Input           []ResponseInput    `json:"input"`
	Reasoning       *ResponseReasoning `json:"reasoning,omitempty"` // Omitted when no effort was requested.
}

// ResponseReasoning configures the provider's reasoning effort.
type ResponseReasoning struct {
	Effort string `json:"effort"`
}

// ResponseInput is one role/content item in a Responses API request.
type ResponseInput struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
