package llm

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Provider represents different LLM providers
type Provider string

const (
	OpenAI    Provider = "openai"
	Anthropic Provider = "anthropic"
	Cohere    Provider = "cohere"
)

// ModelRequest represents a request to an LLM
type ModelRequest struct {
	Prompt     string            `json:"prompt"`
	MaxTokens  int               `json:"max_tokens"`
	Temperature float32          `json:"temperature"`
	Provider   Provider          `json:"provider"`
	ModelName  string           `json:"model_name"`
	Metadata   map[string]string `json:"metadata"`
}

// ModelResponse represents a response from an LLM
type ModelResponse struct {
	Text       string            `json:"text"`
	Provider   Provider          `json:"provider"`
	ModelName  string           `json:"model_name"`
	TokensUsed int              `json:"tokens_used"`
	Latency    time.Duration    `json:"latency"`
	Metadata   map[string]string `json:"metadata"`
}

// ModelRouter handles routing requests to different LLM providers
type ModelRouter struct {
	providers map[Provider]ModelProvider
	mutex     sync.RWMutex
	fallbacks map[string][]string // maps model names to fallback options
}

// ModelProvider interface for different LLM providers
type ModelProvider interface {
	GenerateText(context.Context, ModelRequest) (ModelResponse, error)
	IsAvailable() bool
}

// NewModelRouter creates a new router instance
func NewModelRouter() *ModelRouter {
	return &ModelRouter{
		providers: make(map[Provider]ModelProvider),
		fallbacks: make(map[string][]string),
	}
}

// RegisterProvider adds a new provider to the router
func (r *ModelRouter) RegisterProvider(provider Provider, client ModelProvider) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.providers[provider] = client
}

// SetFallbacks configures fallback models for a given model
func (r *ModelRouter) SetFallbacks(modelName string, fallbackModels []string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.fallbacks[modelName] = fallbackModels
}

// RouteRequest routes the request to appropriate provider with fallback support
func (r *ModelRouter) RouteRequest(ctx context.Context, req ModelRequest) (ModelResponse, error) {
	r.mutex.RLock()
	provider, exists := r.providers[req.Provider]
	r.mutex.RUnlock()

	if !exists {
		return ModelResponse{}, errors.New("provider not found")
	}

	// Try primary model
	if provider.IsAvailable() {
		resp, err := provider.GenerateText(ctx, req)
		if err == nil {
			return resp, nil
		}
	}

	// Try fallbacks
	return r.tryFallbacks(ctx, req)
}

// tryFallbacks attempts to use configured fallback models
func (r *ModelRouter) tryFallbacks(ctx context.Context, req ModelRequest) (ModelResponse, error) {
	r.mutex.RLock()
	fallbacks, exists := r.fallbacks[req.ModelName]
	r.mutex.RUnlock()

	if !exists {
		return ModelResponse{}, errors.New("no fallbacks configured")
	}

	for _, fallbackModel := range fallbacks {
		// Create new request with fallback model
		fallbackReq := req
		fallbackReq.ModelName = fallbackModel

		provider, exists := r.providers[req.Provider]
		if !exists {
			continue
		}

		if provider.IsAvailable() {
			resp, err := provider.GenerateText(ctx, fallbackReq)
			if err == nil {
				return resp, nil
			}
		}
	}

	return ModelResponse{}, errors.New("all fallbacks failed")
}

// Example implementation of an OpenAI provider
type OpenAIProvider struct {
	apiKey     string
	availabile bool
	models     map[string]bool
}

func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		apiKey: apiKey,
		availabile: true,
		models: map[string]bool{
			"gpt-4": true,
			"gpt-3.5-turbo": true,
		},
	}
}

func (p *OpenAIProvider) GenerateText(ctx context.Context, req ModelRequest) (ModelResponse, error) {
	// Implementation of OpenAI API call would go here
	return ModelResponse{
		Text:      "Sample response",
		Provider:  OpenAI,
		ModelName: req.ModelName,
	}, nil
}

func (p *OpenAIProvider) IsAvailable() bool {
	return p.availabile
}

// Usage example
func Example() {
	router := NewModelRouter()

	// Register providers
	router.RegisterProvider(OpenAI, NewOpenAIProvider("api-key"))

	// Configure fallbacks
	router.SetFallbacks("gpt-4", []string{"gpt-3.5-turbo"})

	// Make a request
	req := ModelRequest{
		Prompt:     "Tell me a joke",
		MaxTokens:  100,
		Provider:   OpenAI,
		ModelName:  "gpt-4",
		Temperature: 0.7,
	}

	ctx := context.Background()
	resp, err := router.RouteRequest(ctx, req)
	if err != nil {
		// Handle error
	}

	// Use response
	_ = resp
}
