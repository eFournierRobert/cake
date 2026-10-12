package oai

import (
	"context"
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
