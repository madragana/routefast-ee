// Package rag adds an advisory Retrieval-Augmented Generation copilot to the
// RouteFast EE control plane.
//
// It indexes the append-only audit log into a pgvector store (see
// migrations/004) and answers natural-language questions about a customer's
// fleet, grounded in that customer's own audit records, with citations.
//
// It is strictly read-only and advisory. It never issues credentials, changes
// policy, mutates the audit log, or takes any action. Retrieval is always
// scoped to a single customer_id. The feature is off by default (RAG_ENABLED).
package rag

import (
	"os"
	"strconv"
	"strings"
)

// Backend identifies an inference provider for embeddings or generation.
type Backend string

const (
	BackendOllama    Backend = "ollama"    // local, default (air-gapped friendly)
	BackendOpenAI    Backend = "openai"    // OpenAI cloud API
	BackendAnthropic Backend = "anthropic" // Anthropic cloud API (generation only)
)

// Config controls the RAG copilot. Populated from RAG_* env vars by
// ConfigFromEnv, with local-first defaults.
type Config struct {
	Enabled      bool // RAG_ENABLED (default false)
	TopK         int  // RAG_TOP_K — audit records retrieved per question (default 6)
	ReindexBatch int  // RAG_REINDEX_BATCH — rows embedded per batch (default 128)

	// Embedding backend.
	EmbedBackend Backend // RAG_EMBED_BACKEND (default ollama)
	EmbedModel   string  // RAG_EMBED_MODEL
	EmbedBaseURL string  // RAG_EMBED_BASE_URL
	EmbedAPIKey  string  // RAG_EMBED_API_KEY
	EmbedDim     int     // RAG_EMBED_DIM — must match vector(N) in migration 004 (default 768)

	// Generation (LLM) backend.
	LLMBackend   Backend // RAG_LLM_BACKEND (default ollama)
	LLMModel     string  // RAG_LLM_MODEL
	LLMBaseURL   string  // RAG_LLM_BASE_URL
	LLMAPIKey    string  // RAG_LLM_API_KEY
	LLMMaxTokens int     // RAG_LLM_MAX_TOKENS (default 1024)
}

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func envInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// ConfigFromEnv builds a Config from RAG_* environment variables.
func ConfigFromEnv() Config {
	c := Config{
		Enabled:      envBool("RAG_ENABLED", false),
		TopK:         envInt("RAG_TOP_K", 6),
		ReindexBatch: envInt("RAG_REINDEX_BATCH", 128),

		EmbedBackend: Backend(env("RAG_EMBED_BACKEND", string(BackendOllama))),
		EmbedModel:   env("RAG_EMBED_MODEL", ""),
		EmbedBaseURL: env("RAG_EMBED_BASE_URL", ""),
		EmbedAPIKey:  env("RAG_EMBED_API_KEY", ""),
		EmbedDim:     envInt("RAG_EMBED_DIM", 768),

		LLMBackend:   Backend(env("RAG_LLM_BACKEND", string(BackendOllama))),
		LLMModel:     env("RAG_LLM_MODEL", ""),
		LLMBaseURL:   env("RAG_LLM_BASE_URL", ""),
		LLMAPIKey:    env("RAG_LLM_API_KEY", ""),
		LLMMaxTokens: envInt("RAG_LLM_MAX_TOKENS", 1024),
	}

	if c.EmbedModel == "" {
		switch c.EmbedBackend {
		case BackendOpenAI:
			c.EmbedModel = "text-embedding-3-small"
		default:
			c.EmbedModel = "nomic-embed-text"
		}
	}
	if c.LLMModel == "" {
		switch c.LLMBackend {
		case BackendOpenAI:
			c.LLMModel = "gpt-4o-mini"
		case BackendAnthropic:
			c.LLMModel = "claude-3-5-haiku-latest"
		default:
			c.LLMModel = "llama3.1"
		}
	}
	if c.TopK <= 0 {
		c.TopK = 6
	}
	if c.ReindexBatch <= 0 {
		c.ReindexBatch = 128
	}
	if c.EmbedDim <= 0 {
		c.EmbedDim = 768
	}
	if c.LLMMaxTokens <= 0 {
		c.LLMMaxTokens = 1024
	}
	return c
}
