package openai

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
)

// Client represents an OpenAI client
type Client struct {
    BaseURL    string
    APIKey     string
    Model      string
}

// NewClient creates a new OpenAI client
func NewClient() (*Client, error) {
    baseURL := os.Getenv("OPENAI_BASE_URL")
    if baseURL == "" {
        return nil, fmt.Errorf("OPENAI_BASE_URL environment variable is not set")
    }

    apiKey := os.Getenv("OPENAI_KEY")
    if apiKey == "" {
        return nil, fmt.Errorf("OPENAI_KEY environment variable is not set")
    }

    model := os.Getenv("OPENAI_MODEL")
    if model == "" {
        return nil, fmt.Errorf("OPENAI_MODEL environment variable is not set")
    }

    return &Client{
        BaseURL: baseURL,
        APIKey:  apiKey,
        Model:   model,
    }, nil
}

// ChatCompletionMessage represents a message in the chat completion
type ChatCompletionMessage struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

// ChatCompletionRequest represents a request for chat completion
type ChatCompletionRequest struct {
    Model    string                  `json:"model"`
    Messages []ChatCompletionMessage `json:"messages"`
    MaxTokens int                    `json:"max_tokens,omitempty"`
}

// ChatCompletionResponse represents a response from chat completion
type ChatCompletionResponse struct {
    ID      string `json:"id"`
    Object  string `json:"object"`
    Created int64  `json:"created"`
    Model   string `json:"model"`
    Choices []struct {
        Message ChatCompletionMessage `json:"message"`
    } `json:"choices"`
    Usage struct {
        PromptTokens     int `json:"prompt_tokens"`
        CompletionTokens int `json:"completion_tokens"`
        TotalTokens      int `json:"total_tokens"`
    } `json:"usage"`
}

// CreateChatCompletion creates a chat completion
func (c *Client) CreateChatCompletion(request *ChatCompletionRequest) (*ChatCompletionResponse, error) {
    url := fmt.Sprintf("%s/chat/completions", c.BaseURL)

    jsonData, err := json.Marshal(request)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request: %w", err)
    }

    httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, fmt.Errorf("failed to create HTTP request: %w", err)
    }

    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

    client := &http.Client{}
    resp, err := client.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("failed to send request to OpenAI: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("OpenAI service returned non-OK status: %s - %s", resp.Status, string(body))
    }

    var response ChatCompletionResponse
    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }

    return &response, nil
}
