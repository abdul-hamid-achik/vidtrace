package embed

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOllamaEmbedReturnsVectorsAndProfile(t *testing.T) {
	var gotReq ollamaEmbedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embed" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			t.Errorf("decode request: %v", err)
		}
		vecs := make([][]float32, len(gotReq.Input))
		for i := range gotReq.Input {
			vecs[i] = []float32{float32(i), 0.5, -0.5}
		}
		_ = json.NewEncoder(w).Encode(ollamaEmbedResponse{Embeddings: vecs})
	}))
	defer server.Close()

	embedder := NewOllama(server.URL, "nomic-embed-text")
	vecs, err := embedder.Embed(context.Background(), []string{"first", "second"})
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}
	if len(vecs) != 2 || len(vecs[0]) != 3 {
		t.Fatalf("unexpected vectors: %#v", vecs)
	}
	if gotReq.Model != "nomic-embed-text" || len(gotReq.Input) != 2 {
		t.Fatalf("unexpected request: %#v", gotReq)
	}

	profile := embedder.Profile()
	if profile.Provider != ProviderOllama || profile.Model != "nomic-embed-text" || profile.Dimensions != 3 {
		t.Fatalf("unexpected profile: %#v", profile)
	}
}

func TestOllamaEmbedRejectsEmptyModel(t *testing.T) {
	embedder := NewOllama("http://localhost:11434", "")
	if _, err := embedder.Embed(context.Background(), []string{"x"}); err == nil {
		t.Fatal("expected error for empty model")
	}
}

func TestOllamaEmbedReportsServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"model not found"}`))
	}))
	defer server.Close()

	embedder := NewOllama(server.URL, "missing-model")
	if _, err := embedder.Embed(context.Background(), []string{"x"}); err == nil {
		t.Fatal("expected error for server failure")
	}
}

func TestOllamaEmbedRejectsCountMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ollamaEmbedResponse{Embeddings: [][]float32{{1, 2, 3}}})
	}))
	defer server.Close()

	embedder := NewOllama(server.URL, "nomic-embed-text")
	if _, err := embedder.Embed(context.Background(), []string{"a", "b"}); err == nil {
		t.Fatal("expected error when embedding count does not match inputs")
	}
}

func TestOllamaEmbedRejectsRaggedVectors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ollamaEmbedRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		// Return the right count, but a later vector has a different length.
		vecs := make([][]float32, len(req.Input))
		for i := range req.Input {
			if i == 0 {
				vecs[i] = []float32{1, 2, 3}
			} else {
				vecs[i] = []float32{1, 2}
			}
		}
		_ = json.NewEncoder(w).Encode(ollamaEmbedResponse{Embeddings: vecs})
	}))
	defer server.Close()

	embedder := NewOllama(server.URL, "nomic-embed-text")
	if _, err := embedder.Embed(context.Background(), []string{"a", "b"}); err == nil {
		t.Fatal("expected error for inconsistent embedding lengths")
	}
}

func TestOllamaEmbedBatchesLargeInputs(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		var req ollamaEmbedRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if len(req.Input) > ollamaBatchSize {
			t.Errorf("batch of %d exceeds limit %d", len(req.Input), ollamaBatchSize)
		}
		vecs := make([][]float32, len(req.Input))
		for i := range req.Input {
			vecs[i] = []float32{1, 0}
		}
		_ = json.NewEncoder(w).Encode(ollamaEmbedResponse{Embeddings: vecs})
	}))
	defer server.Close()

	texts := make([]string, ollamaBatchSize+5)
	for i := range texts {
		texts[i] = "text"
	}
	embedder := NewOllama(server.URL, "m")
	vecs, err := embedder.Embed(context.Background(), texts)
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}
	if len(vecs) != len(texts) {
		t.Fatalf("expected %d vectors, got %d", len(texts), len(vecs))
	}
	if requests < 2 {
		t.Fatalf("expected the oversized input to be split into multiple requests, got %d", requests)
	}
}

func TestBuild(t *testing.T) {
	if e, err := Build("", "", ""); err != nil || e != nil {
		t.Fatalf("empty provider should yield a nil embedder, got %v %v", e, err)
	}
	if e, err := Build("none", "", ""); err != nil || e != nil {
		t.Fatalf("none provider should yield a nil embedder, got %v %v", e, err)
	}
	if _, err := Build("ollama", "", ""); err == nil {
		t.Fatal("ollama without a model should error")
	}
	if e, err := Build("ollama", "nomic-embed-text", ""); err != nil || e == nil {
		t.Fatalf("ollama with a model should build, got %v %v", e, err)
	}
	if _, err := Build("magic", "m", ""); err == nil {
		t.Fatal("unknown provider should error")
	}
}

func TestNewOllamaDefaultsBaseURL(t *testing.T) {
	embedder := NewOllama("", "m")
	if embedder.BaseURL != defaultOllamaURL {
		t.Fatalf("expected default base URL, got %q", embedder.BaseURL)
	}
	if NewOllama("http://host:1234/", "m").BaseURL != "http://host:1234" {
		t.Fatal("expected trailing slash trimmed")
	}
}
