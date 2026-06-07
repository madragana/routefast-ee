package rag

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

// Embedder turns text into a vector. Implementations must be safe for
// concurrent use.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// newEmbedder selects an embedding client from config. Default is local Ollama.
// Anthropic has no embeddings API, so it is not an option here; keep embeddings
// on ollama or openai when using an Anthropic LLM.
func newEmbedder(c Config) (Embedder, error) {
	httpc := &http.Client{Timeout: 60 * time.Second}
	switch c.EmbedBackend {
	case BackendOllama, "":
		base := c.EmbedBaseURL
		if base == "" {
			base = "http://localhost:11434"
		}
		return &ollamaEmbedder{http: httpc, base: strings.TrimRight(base, "/"), model: c.EmbedModel}, nil
	case BackendOpenAI:
		if c.EmbedAPIKey == "" {
			return nil, fmt.Errorf("rag: RAG_EMBED_API_KEY required for openai embeddings")
		}
		base := c.EmbedBaseURL
		if base == "" {
			base = "https://api.openai.com/v1"
		}
		return &openAIEmbedder{http: httpc, base: strings.TrimRight(base, "/"), model: c.EmbedModel, key: c.EmbedAPIKey}, nil
	default:
		return nil, fmt.Errorf("rag: unsupported embedding backend %q (use ollama or openai)", c.EmbedBackend)
	}
}

func postJSON(ctx context.Context, hc *http.Client, url string, headers map[string]string, payload any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(out)))
	}
	return out, nil
}

type ollamaEmbedder struct {
	http  *http.Client
	base  string
	model string
}

func (e *ollamaEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	out, err := postJSON(ctx, e.http, e.base+"/api/embeddings", nil, map[string]any{
		"model":  e.model,
		"prompt": text,
	})
	if err != nil {
		return nil, err
	}
	var r struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.Unmarshal(out, &r); err != nil {
		return nil, err
	}
	if len(r.Embedding) == 0 {
		return nil, fmt.Errorf("ollama: empty embedding")
	}
	return r.Embedding, nil
}

type openAIEmbedder struct {
	http  *http.Client
	base  string
	model string
	key   string
}

func (e *openAIEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	out, err := postJSON(ctx, e.http, e.base+"/embeddings",
		map[string]string{"Authorization": "Bearer " + e.key},
		map[string]any{"model": e.model, "input": text})
	if err != nil {
		return nil, err
	}
	var r struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &r); err != nil {
		return nil, err
	}
	if len(r.Data) == 0 || len(r.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("openai: empty embedding")
	}
	return r.Data[0].Embedding, nil
}
