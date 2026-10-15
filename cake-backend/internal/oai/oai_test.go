package oai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetModels(t *testing.T) {
	const response = `{"object":"list","data":[{"id":"model-a","object":"model","created":123,"owned_by":"provider"}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1/models", r.URL.Path)
		assert.Equal(t, "Bearer secret-key", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(server.Close)

	got, err := New().GetModels(context.Background(), "secret-key", server.URL+"/v1")
	require.NoError(t, err)
	assert.Equal(t, "list", got.Object)
	require.Len(t, got.Data, 1)
	assert.Equal(t, Model{ID: "model-a", Object: "model", Created: 123, OwnedBy: "provider"}, got.Data[0])
}

func TestGetModelsWithoutAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	t.Cleanup(server.Close)

	got, err := New().GetModels(context.Background(), "", server.URL)
	require.NoError(t, err)
	assert.Empty(t, got.Data)
}

func TestGetModelsRejectsNonSuccessStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	_, err := New().GetModels(context.Background(), "", server.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "503")
}

func TestGetModelsRejectsInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	t.Cleanup(server.Close)

	_, err := New().GetModels(context.Background(), "", server.URL)
	assert.Error(t, err)
}

func TestGetModelsHonorsCanceledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("request should not reach the server")
	}))
	t.Cleanup(server.Close)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := New().GetModels(ctx, "", server.URL)
	assert.Error(t, err)
}

func TestCreateResponseSendsExpectedRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/responses", r.URL.Path)
		assert.Equal(t, "Bearer secret-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "text/event-stream", r.Header.Get("Accept"))

		var body ResponseRequestBody
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "model-a", body.Model)
		assert.Equal(t, "Be concise", body.Instructions)
		assert.False(t, body.Stream)
		assert.False(t, body.Store)
		assert.Equal(t, 8192, body.MaxOutputTokens)
		assert.Equal(t, []ResponseInput{{Role: "user", Content: "Hi"}}, body.Input)
		require.NotNil(t, body.Reasoning)
		assert.Equal(t, "medium", body.Reasoning.Effort)
		_, _ = w.Write([]byte(`{"output":[{"type":"message","content":[{"type":"output_text","text":"Hello"}]}]}`))
	}))
	t.Cleanup(server.Close)

	response, err := New().CreateResponse(context.Background(), "secret-key", server.URL+"/v1", "model-a", "medium", "Be concise", []ResponseInput{{Role: "user", Content: "Hi"}})
	require.NoError(t, err)
	require.Len(t, response.Output, 1)
	require.Len(t, response.Output[0].Content, 1)
	assert.Equal(t, "Hello", response.Output[0].Content[0].Text)
}

func TestCreateResponseRejectsNonSuccessStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	_, err := New().CreateResponse(context.Background(), "", server.URL, "model-a", "", "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "503")
}

func TestCreateStreamingResponseForwardsDeltasAndTerminalEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Bearer secret-key", r.Header.Get("Authorization"))
		var body ResponseRequestBody
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.True(t, body.Stream)
		assert.Equal(t, "model-a", body.Model)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"Hello\"}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.completed\"}\n\n"))
	}))
	t.Cleanup(server.Close)

	events, err := New().CreateStreamingResponse(context.Background(), "secret-key", server.URL, "model-a", "", "", nil)
	require.NoError(t, err)
	var got []ResponseEvent
	for event := range events {
		got = append(got, event)
	}
	require.Len(t, got, 2)
	assert.Equal(t, "response.output_text.delta", got[0].Data.Type)
	assert.Equal(t, "Hello", got[0].Data.Delta)
	assert.NoError(t, got[0].Error)
	assert.Equal(t, "response.completed", got[1].Data.Type)
	assert.NoError(t, got[1].Error)
}

func TestCreateStreamingResponseReportsMalformedSSEData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: not-json\n\n"))
	}))
	t.Cleanup(server.Close)

	events, err := New().CreateStreamingResponse(context.Background(), "", server.URL, "model-a", "", "", nil)
	require.NoError(t, err)
	event, ok := <-events
	require.True(t, ok)
	require.Error(t, event.Error)
	assert.Contains(t, event.Error.Error(), "unmarshal SSE data")
}

func TestCreateStreamingResponseRejectsNonSuccessStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	_, err := New().CreateStreamingResponse(context.Background(), "", server.URL, "model-a", "", "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "503")
}
