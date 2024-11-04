package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// ProxyConfig holds configuration for the LLM proxy service
type ProxyConfig struct {
	APIKey          string
	CacheEnabled    bool
	RetryConfig     RetryConfig
	RateLimit       int           // requests per minute
	CostLimit       float64       // maximum cost per day
	CustomHeaders   http.Header
	FilterFunction  func(string) string // for PII filtering
}

type RetryConfig struct {
	MaxRetries  int
	BackoffBase time.Duration
}

// ProxyMetrics tracks usage and performance metrics
type ProxyMetrics struct {
	TotalRequests     int64
	CacheHits         int64
	Latency          time.Duration
	TokensUsed       int64
	EstimatedCost    float64
	mu               sync.RWMutex
}

// LLMProxy provides a proxy layer for LLM requests
type LLMProxy struct {
	config  ProxyConfig
	metrics ProxyMetrics
	cache   *RequestCache
}

// RequestCache implements a simple cache for LLM requests
type RequestCache struct {
	entries sync.Map // map[string]CacheEntry
}

type CacheEntry struct {
	Response    json.RawMessage
	Expiration  time.Time
	TokensUsed  int
	Cost        float64
}

// NewLLMProxy creates a new proxy instance
func NewLLMProxy(config ProxyConfig) *LLMProxy {
	return &LLMProxy{
		config: config,
		cache:  &RequestCache{},
		metrics: ProxyMetrics{},
	}
}

// Request represents an LLM API request with metadata
type Request struct {
	Prompt       string                 `json:"prompt"`
	Model        string                 `json:"model"`
	Temperature  float32                `json:"temperature"`
	MaxTokens    int                    `json:"max_tokens"`
	Metadata     map[string]interface{} `json:"metadata"`
	CacheKey     string                 `json:"cache_key,omitempty"`
	RequestID    string                 `json:"request_id"`
	UserID       string                 `json:"user_id,omitempty"`
}

// ProxyResponse wraps the LLM response with additional metadata
type ProxyResponse struct {
	Text         string                 `json:"text"`
	TokensUsed   int                    `json:"tokens_used"`
	Cost         float64                `json:"cost"`
	CacheHit     bool                   `json:"cache_hit"`
	Latency      time.Duration          `json:"latency"`
	Model        string                 `json:"model"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// ProcessRequest handles an LLM request through the proxy
func (p *LLMProxy) ProcessRequest(ctx context.Context, req Request) (*ProxyResponse, error) {
	start := time.Now()

	// Check rate limits
	if err := p.checkRateLimits(); err != nil {
		return nil, err
	}

	// Filter sensitive data if configured
	if p.config.FilterFunction != nil {
		req.Prompt = p.config.FilterFunction(req.Prompt)
	}

	// Try cache first if enabled
	if p.config.CacheEnabled {
		if cached, hit := p.checkCache(req.CacheKey); hit {
			p.recordMetrics(0, cached.TokensUsed, cached.Cost, true)
			return &ProxyResponse{
				Text:      string(cached.Response),
				CacheHit: true,
				TokensUsed: cached.TokensUsed,
				Cost:     cached.Cost,
				Latency:  time.Since(start),
			}, nil
		}
	}

	// Process request with retries
	response, err := p.makeRequestWithRetries(ctx, req)
	if err != nil {
		return nil, err
	}

	// Update metrics
	p.recordMetrics(time.Since(start), response.TokensUsed, response.Cost, false)

	// Cache response if enabled
	if p.config.CacheEnabled {
		p.cacheResponse(req.CacheKey, response)
	}

	return response, nil
}

// checkRateLimits ensures we're within configured limits
func (p *LLMProxy) checkRateLimits() error {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()

	if p.metrics.EstimatedCost >= p.config.CostLimit {
		return fmt.Errorf("daily cost limit exceeded: %.2f", p.config.CostLimit)
	}

	return nil
}

// recordMetrics updates usage metrics
func (p *LLMProxy) recordMetrics(latency time.Duration, tokens int64, cost float64, cacheHit bool) {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()

	p.metrics.TotalRequests++
	if cacheHit {
		p.metrics.CacheHits++
	}
	p.metrics.Latency += latency
	p.metrics.TokensUsed += tokens
	p.metrics.EstimatedCost += cost
}

// makeRequestWithRetries implements retry logic with exponential backoff
func (p *LLMProxy) makeRequestWithRetries(ctx context.Context, req Request) (*ProxyResponse, error) {
	var lastErr error

	for attempt := 0; attempt < p.config.RetryConfig.MaxRetries; attempt++ {
		response, err := p.makeRequest(ctx, req)
		if err == nil {
			return response, nil
		}

		lastErr = err
		backoff := p.config.RetryConfig.BackoffBase * time.Duration(1<<attempt)
		
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
			continue
		}
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// Example usage
func Example() {
	config := ProxyConfig{
		APIKey:       "your-api-key",
		CacheEnabled: true,
		RetryConfig: RetryConfig{
			MaxRetries:  3,
			BackoffBase: time.Second,
		},
		RateLimit:  100,
		CostLimit:  50.0,
		FilterFunction: func(prompt string) string {
			// Implement PII filtering logic
			return prompt
		},
	}

	proxy := NewLLMProxy(config)

	req := Request{
		Prompt:    "Explain quantum computing",
		Model:     "gpt-4",
		MaxTokens: 100,
		RequestID: "req-123",
		UserID:    "user-456",
		Metadata: map[string]interface{}{
			"purpose": "education",
			"source":  "web-app",
		},
	}

	ctx := context.Background()
	resp, err := proxy.ProcessRequest(ctx, req)
	if err != nil {
		// Handle error
	}

	// Use response
	fmt.Printf("Response: %+v\n", resp)
}
