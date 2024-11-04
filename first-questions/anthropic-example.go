package main

import (
	"bytes"
	"io"
	"net/http"
)

const baseUrl string = "https://api.anthropic.com/v1/messages"

type anthropicClient struct {
	apiKey string
	client *http.Client
}

func newAnthropicClient(apiKey string) *anthropicClient {
	return &anthropicClient{
		apiKey: apiKey,
		client: &http.Client{},
	}

}

func (c *anthropicClient) generate2(prompt string) (string, error) {
	body := []byte(`{
		"model": "claude-3-5-sonnet-20240620",
		"max_tokens": 1024,
		"messages": [
			{
				"role": "user",
				"content": [
					{
						"type": "text",
						"text": "` + prompt + `"
					}
				]
			}
		]
	}`)

	// log.Println(string(body))

	req, _ := http.NewRequest("POST", baseUrl, bytes.NewBuffer(body))

	req.Header.Add("x-api-key", c.apiKey)
	req.Header.Add("anthropic-version", "2023-06-01")
	req.Header.Add("Content-Type", "application/json")

	res, err := c.client.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	readBody, _ := io.ReadAll(res.Body)
	return string(readBody), nil
}
