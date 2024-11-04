package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// LLMResponse represents a typical response from an LLM API
type LLMResponse struct {
	ID        string    `json:"id"`
	Created   time.Time `json:"created"`
	Choices   []Choice  `json:"choices"`
	Usage     Usage     `json:"usage"`
}

type Choice struct {
	Text  string `json:"text"`
	Index int    `json:"index"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Example 1: Concurrent API requests with goroutines and channels
func processBatchPrompts(prompts []string) []LLMResponse {
	responses := make([]LLMResponse, len(prompts))
	var wg sync.WaitGroup
	
	// Create buffered channel to control concurrency
	semaphore := make(chan struct{}, 5) // Limit to 5 concurrent requests
	
	for i, prompt := range prompts {
		wg.Add(1)
		go func(index int, prompt string) {
			defer wg.Done()
			
			semaphore <- struct{}{} // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore
			
			// Make API request with context and timeout
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			
			response, err := makeAPIRequest(ctx, prompt)
			if err == nil {
				responses[index] = response
			}
		}(i, prompt)
	}
	
	wg.Wait()
	return responses
}

// Example 2: Robust HTTP client with retry logic
func makeAPIRequest(ctx context.Context, prompt string) (LLMResponse, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "POST", "https://api.llm-service.com/v1/completions", nil)
		if err != nil {
			continue
		}
		
		resp, err := client.Do(req)
		if err != nil {
			// Implement exponential backoff
			time.Sleep(time.Duration(attempt*attempt) * time.Second)
			continue
		}
		defer resp.Body.Close()
		
		if resp.StatusCode == http.StatusOK {
			var llmResponse LLMResponse
			if err := json.NewDecoder(resp.Body).Decode(&llmResponse); err == nil {
				return llmResponse, nil
			}
		}
	}
	return LLMResponse{}, fmt.Errorf("failed after %d retries", maxRetries)
}

// Example 3: Streaming response handler
func handleStreamingResponse(ctx context.Context, prompt string) (<-chan string, <-chan error) {
	resultChan := make(chan string)
	errChan := make(chan error, 1)
	
	go func() {
		defer close(resultChan)
		defer close(errChan)
		
		req, err := http.NewRequestWithContext(ctx, "POST", "https://api.llm-service.com/v1/stream", nil)
		if err != nil {
			errChan <- err
			return
		}
		
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			errChan <- err
			return
		}
		defer resp.Body.Close()
		
		decoder := json.NewDecoder(resp.Body)
		for {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
				var chunk struct {
					Text string `json:"text"`
				}
				if err := decoder.Decode(&chunk); err != nil {
					errChan <- err
					return
				}
				resultChan <- chunk.Text
			}
		}
	}()
	
	return resultChan, errChan
}

func main() {
	// Example usage of concurrent processing
	prompts := []string{
		"What is the capital of France?",
		"What is the largest planet?",
		"Who wrote Romeo and Juliet?",
	}
	
	responses := processBatchPrompts(prompts)
	for i, resp := range responses {
		fmt.Printf("Response %d: %+v\n", i, resp)
	}
	
	// Example usage of streaming response
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	resultChan, errChan := handleStreamingResponse(ctx, "Tell me a story...")
	for {
		select {
		case chunk, ok := <-resultChan:
			if !ok {
				return
			}
			fmt.Print(chunk)
		case err := <-errChan:
			fmt.Printf("Error: %v\n", err)
			return
		}
	}
}
