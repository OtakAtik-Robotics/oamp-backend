package llm

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

// LLMProvider defines the interface for LLM API providers
type LLMProvider interface {
	GenerateHealthAnalysis(prompt string) (string, error)
}

// Config holds common LLM configuration
type Config struct {
	Provider string
	APIKey   string
	Model    string
	BaseURL  string
	GroupID  string // For Minimax only
}

// Shared HTTP client with connection pooling
var sharedClient = &http.Client{
	Timeout: 120 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        10,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
	},
}

var (
	cachedProvider LLMProvider
	cachedErr      error
	once           sync.Once
)

// NewProvider creates an LLM provider based on AI_PROVIDER env var.
// The provider is cached after first call (initialized once via sync.Once).
func NewProvider() (LLMProvider, error) {
	once.Do(func() {
		cachedProvider, cachedErr = newProviderImpl()
	})
	return cachedProvider, cachedErr
}

func newProviderImpl() (LLMProvider, error) {
	cfg := Config{
		Provider: os.Getenv("AI_PROVIDER"),
		APIKey:   os.Getenv("AI_API_KEY"),
		Model:    os.Getenv("AI_MODEL"),
		BaseURL:  os.Getenv("AI_BASE_URL"),
		GroupID:  os.Getenv("MINIMAX_GROUP_ID"),
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("AI_API_KEY not configured")
	}
	if cfg.Provider == "" {
		return nil, fmt.Errorf("AI_PROVIDER not configured (openai, gemini, claude, minimax)")
	}

	switch cfg.Provider {
	case "openai":
		return newOpenAIProvider(cfg), nil
	case "gemini":
		return newGeminiProvider(cfg), nil
	case "claude":
		return newClaudeProvider(cfg), nil
	case "minimax":
		return newMinimaxProvider(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported AI_PROVIDER: %s", cfg.Provider)
	}
}
