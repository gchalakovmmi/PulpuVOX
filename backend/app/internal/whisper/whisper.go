package whisper

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "mime/multipart"
    "net/http"
    "os"
)

// TranscribeService handles audio transcription
type TranscribeService struct {
    WhisperURL string
    Provider   string
    APIKey     string
    Model      string
}

// NewTranscribeService creates a new transcription service
func NewTranscribeService() (*TranscribeService, error) {
    whisperURL := os.Getenv("WHISPER_URL")
    if whisperURL == "" {
        return nil, fmt.Errorf("WHISPER_URL environment variable is not set")
    }

    provider := os.Getenv("WHISPER_PROVIDER")
    if provider == "" {
        provider = "docker" // Default to docker provider
    }

    apiKey := os.Getenv("WHISPER_KEY")
    model := os.Getenv("WHISPER_MODEL")

    return &TranscribeService{
        WhisperURL: whisperURL,
        Provider:   provider,
        APIKey:     apiKey,
        Model:      model,
    }, nil
}

// TranscribeRequest represents a transcription request
type TranscribeRequest struct {
    AudioData    []byte
    FileName     string
    Language     string
    Task         string
    OutputFormat string
}

// TranscribeResponse represents a transcription response
type TranscribeResponse struct {
    Text     string `json:"text"`
    Language string `json:"language"`
    Error    string `json:"error,omitempty"`
}

// SendToWhisper sends audio data to the Whisper service for transcription
func (ts *TranscribeService) SendToWhisper(req *TranscribeRequest) (*TranscribeResponse, error) {
    body := &bytes.Buffer{}
    writer := multipart.NewWriter(body)

    part, err := writer.CreateFormFile("file", req.FileName)
    if err != nil {
        return nil, fmt.Errorf("failed to create form file: %w", err)
    }

    if _, err := io.Copy(part, bytes.NewReader(req.AudioData)); err != nil {
        return nil, fmt.Errorf("failed to write audio data to form: %w", err)
    }

    if err := writer.Close(); err != nil {
        return nil, fmt.Errorf("failed to close multipart writer: %w", err)
    }

    // Build the URL with query parameters for Docker provider
    // For Groq, we don't need to add query parameters because they use the OpenAI API format
    url := ts.WhisperURL
    if ts.Provider == "docker" {
        url = fmt.Sprintf("%s?task=%s&language=%s&output=%s", ts.WhisperURL, req.Task, req.Language, req.OutputFormat)
    }

    httpReq, err := http.NewRequest("POST", url, body)
    if err != nil {
        return nil, fmt.Errorf("failed to create HTTP request: %w", err)
    }

    httpReq.Header.Set("Content-Type", writer.FormDataContentType())
    if ts.APIKey != "" {
        httpReq.Header.Set("Authorization", "Bearer "+ts.APIKey)
    }

    client := &http.Client{}
    resp, err := client.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("failed to send request to Whisper: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("Whisper service returned non-OK status: %s - %s", resp.Status, string(body))
    }

    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read response body: %w", err)
    }

    var result TranscribeResponse
    if err := json.Unmarshal(respBody, &result); err != nil {
        return nil, fmt.Errorf("failed to parse Whisper response: %w", err)
    }

    return &result, nil
}
