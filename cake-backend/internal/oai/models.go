// Package oai provides the small OpenAI-compatible API client used to fetch
// provider model catalogs.
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

type ResponseEvent struct {
	Data  ResponseEventData
	Error error
}

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

type ResponseEventData struct {
	Type  string `json:"type"`
	Delta string `json:"delta"`
}

type ResponseRequestBody struct {
	Model           string             `json:"model"`
	Instructions    string             `json:"instructions"`
	Stream          bool               `json:"stream"`
	Store           bool               `json:"store"`
	MaxOutputTokens int                `json:"max_output_tokens"`
	Input           []ResponseInput    `json:"input"`
	Reasoning       *ResponseReasoning `json:"reasoning,omitempty"`
}

type ResponseReasoning struct {
	Effort string `json:"effort"`
}

type ResponseInput struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
