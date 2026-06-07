package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Generator turns a system + user prompt into a single text completion. Safe
// for concurrent use.
type Generator interface {
	Generate(ctx context.Context, system, user string) (string, error)
}

// newGenerator selects an LLM client from config. Default is local Ollama.
func newGenerator(c Config) (Generator, error) {
	httpc := &http.Client{Timeout: 60 * time.Second}
	switch c.LLMBackend {
	case BackendOllama, "":
		base := c.LLMBaseURL
		if base == "" {
			base = "http://localhost:11434"
		}
		return &ollamaGen{http: httpc, base: strings.TrimRight(base, "/"), model: c.LLMModel}, nil
	case BackendOpenAI:
		if c.LLMAPIKey == "" {
			return nil, fmt.Errorf("rag: RAG_LLM_API_KEY required for openai generation")
		}
		base := c.LLMBaseURL
		if base == "" {
			base = "https://api.openai.com/v1"
		}
		return &openAIGen{http: httpc, base: strings.TrimRight(base, "/"), model: c.LLMModel, key: c.LLMAPIKey, maxTokens: c.LLMMaxTokens}, nil
	case BackendAnthropic:
		if c.LLMAPIKey == "" {
			return nil, fmt.Errorf("rag: RAG_LLM_API_KEY required for anthropic generation")
		}
		base := c.LLMBaseURL
		if base == "" {
			base = "https://api.anthropic.com/v1"
		}
		return &anthropicGen{http: httpc, base: strings.TrimRight(base, "/"), model: c.LLMModel, key: c.LLMAPIKey, maxTokens: c.LLMMaxTokens}, nil
	default:
		return nil, fmt.Errorf("rag: unsupported llm backend %q (use ollama, openai, or anthropic)", c.LLMBackend)
	}
}

type ollamaGen struct {
	http  *http.Client
	base  string
	model string
}

func (g *ollamaGen) Generate(ctx context.Context, system, user string) (string, error) {
	out, err := postJSON(ctx, g.http, g.base+"/api/chat", nil, map[string]any{
		"model":  g.model,
		"stream": false,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
	})
	if err != nil {
		return "", err
	}
	var r struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(out, &r); err != nil {
		return "", err
	}
	return strings.TrimSpace(r.Message.Content), nil
}

type openAIGen struct {
	http      *http.Client
	base      string
	model     string
	key       string
	maxTokens int
}

func (g *openAIGen) Generate(ctx context.Context, system, user string) (string, error) {
	out, err := postJSON(ctx, g.http, g.base+"/chat/completions",
		map[string]string{"Authorization": "Bearer " + g.key},
		map[string]any{
			"model": g.model,
			"messages": []map[string]string{
				{"role": "system", "content": system},
				{"role": "user", "content": user},
			},
			"max_tokens": g.maxTokens,
		})
	if err != nil {
		return "", err
	}
	var r struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(out, &r); err != nil {
		return "", err
	}
	if len(r.Choices) == 0 {
		return "", fmt.Errorf("openai: empty response")
	}
	return strings.TrimSpace(r.Choices[0].Message.Content), nil
}

type anthropicGen struct {
	http      *http.Client
	base      string
	model     string
	key       string
	maxTokens int
}

func (g *anthropicGen) Generate(ctx context.Context, system, user string) (string, error) {
	out, err := postJSON(ctx, g.http, g.base+"/messages",
		map[string]string{"x-api-key": g.key, "anthropic-version": "2023-06-01"},
		map[string]any{
			"model":      g.model,
			"max_tokens": g.maxTokens,
			"system":     system,
			"messages": []map[string]any{
				{"role": "user", "content": user},
			},
		})
	if err != nil {
		return "", err
	}
	var r struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(out, &r); err != nil {
		return "", err
	}
	var sb strings.Builder
	for _, b := range r.Content {
		if b.Type == "text" {
			sb.WriteString(b.Text)
		}
	}
	return strings.TrimSpace(sb.String()), nil
}
