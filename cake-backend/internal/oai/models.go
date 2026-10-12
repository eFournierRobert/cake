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
