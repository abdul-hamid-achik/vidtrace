// Package embed provides optional text-embedding providers for semantic and
// hybrid evidence search. Providers are pluggable behind the Embedder interface
// so the rest of vidtrace stays independent of any specific embedding backend,
// and so tests can run without a live provider.
package embed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Provider identifiers for embedders.
const (
	ProviderOllama = "ollama"
)

const defaultOllamaURL = "http://localhost:11434"

// Profile describes the embedding configuration used to build a semantic index.
// It is stored alongside the index so search can reject a mismatched embedder.
type Profile struct {
	Provider   string `json:"provider"`
	Model      string `json:"model"`
	Dimensions int    `json:"dimensions"`
}

// Embedder converts text into vectors. Implementations must return one vector
// per input, in order, all with the same dimension.
type Embedder interface {
	// Embed returns one embedding per input text, preserving order.
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	// Profile reports the provider and model. Dimensions may be zero until the
	// first successful Embed call has observed the vector length.
	Profile() Profile
}

// Ollama is an Embedder backed by a local Ollama server's embeddings endpoint.
// It orchestrates an external tool over HTTP, matching how vidtrace shells out
// to ffmpeg, ffprobe, tesseract, and whisper.
type Ollama struct {
	BaseURL string
	Model   string
	Client  *http.Client

	dims int
}

// NewOllama builds an Ollama embedder. An empty baseURL defaults to the local
// Ollama server, and an empty model is rejected later by Embed.
func NewOllama(baseURL, model string) *Ollama {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		trimmed = defaultOllamaURL
	}
	return &Ollama{
		BaseURL: strings.TrimRight(trimmed, "/"),
		Model:   strings.TrimSpace(model),
		Client:  &http.Client{Timeout: 60 * time.Second},
	}
}

type ollamaEmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type ollamaEmbedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
	Error      string      `json:"error"`
}

// Embed calls the Ollama /api/embed batch endpoint.
func (o *Ollama) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	if strings.TrimSpace(o.Model) == "" {
		return nil, fmt.Errorf("embedding model is required")
	}

	payload, err := json.Marshal(ollamaEmbedRequest{Model: o.Model, Input: texts})
	if err != nil {
		return nil, fmt.Errorf("encode ollama request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.BaseURL+"/api/embed", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := o.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call ollama at %s: %w", o.BaseURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read ollama response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama embed failed (%s): %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var decoded ollamaEmbedResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, fmt.Errorf("decode ollama response: %w", err)
	}
	if decoded.Error != "" {
		return nil, fmt.Errorf("ollama: %s", decoded.Error)
	}
	if len(decoded.Embeddings) != len(texts) {
		return nil, fmt.Errorf("ollama returned %d embeddings for %d inputs", len(decoded.Embeddings), len(texts))
	}
	if len(decoded.Embeddings[0]) == 0 {
		return nil, fmt.Errorf("ollama returned empty embeddings")
	}
	o.dims = len(decoded.Embeddings[0])
	return decoded.Embeddings, nil
}

// Profile reports the Ollama provider and model. Dimensions is populated after
// the first successful Embed call.
func (o *Ollama) Profile() Profile {
	return Profile{Provider: ProviderOllama, Model: o.Model, Dimensions: o.dims}
}
