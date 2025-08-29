package whisper

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "mime/multipart"
    "net/http"
    "os"
    "runtime"
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
    Model        string // Optional: override default model
}

// TranscribeResponse represents a transcription response
type TranscribeResponse struct {
    Text     string `json:"text"`
    Language string `json:"language"`
    Error    string `json:"error,omitempty"`
}

// getCallerInfo returns file and line information for error reporting
func getCallerInfo() string {
    pc, file, line, ok := runtime.Caller(2) // Skip 2 frames to get the actual caller
    if ok {
        return fmt.Sprintf("at %s:%d (%s)", file, line, runtime.FuncForPC(pc).Name())
    }
    return "at unknown location"
}

// SendToWhisper sends audio data to the Whisper service for transcription
func (ts *TranscribeService) SendToWhisper(req *TranscribeRequest) (*TranscribeResponse, error) {
    callerInfo := getCallerInfo()
    
    switch ts.Provider {
    case "groq":
        return ts.sendToGroq(req, callerInfo)
    case "docker":
        fallthrough
    default:
        return ts.sendToDocker(req, callerInfo)
    }
}

// sendToDocker sends request to the Docker container
func (ts *TranscribeService) sendToDocker(req *TranscribeRequest, callerInfo string) (*TranscribeResponse, error) {
    body := &bytes.Buffer{}
    writer := multipart.NewWriter(body)
    part, err := writer.CreateFormFile("audio_file", req.FileName)
    if err != nil {
        return nil, fmt.Errorf("failed to create form file %s: %w", callerInfo, err)
    }
    if _, err := io.Copy(part, bytes.NewReader(req.AudioData)); err != nil {
        return nil, fmt.Errorf("failed to write audio data to form %s: %w", callerInfo, err)
    }
    if err := writer.Close(); err != nil {
        return nil, fmt.Errorf("failed to close multipart writer %s: %w", callerInfo, err)
    }

    // Build the URL with query parameters
    whisperURL := fmt.Sprintf("%s?encode=true&task=%s&language=%s&output=%s",
        ts.WhisperURL, req.Task, req.Language, req.OutputFormat)
    
    httpReq, err := http.NewRequest("POST", whisperURL, body)
    if err != nil {
        return nil, fmt.Errorf("failed to create HTTP request %s: %w", callerInfo, err)
    }
    httpReq.Header.Set("Content-Type", writer.FormDataContentType())

    client := &http.Client{}
    resp, err := client.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("failed to send request to Whisper %s: %w", callerInfo, err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("Whisper service returned error %s: Status %d, Body: %s", 
            callerInfo, resp.StatusCode, string(body))
    }

    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read response body %s: %w", callerInfo, err)
    }

    var result TranscribeResponse
    if err := json.Unmarshal(respBody, &result); err != nil {
        return nil, fmt.Errorf("failed to parse Whisper response %s: %w", callerInfo, err)
    }

    return &result, nil
}

// sendToGroq sends request to Groq API
func (ts *TranscribeService) sendToGroq(req *TranscribeRequest, callerInfo string) (*TranscribeResponse, error) {
    body := &bytes.Buffer{}
    writer := multipart.NewWriter(body)

    // Use the model from the service (set via env) unless overridden in the request
    model := ts.Model
    if req.Model != "" {
        model = req.Model
    }
    if model == "" {
        return nil, fmt.Errorf("model is required for Groq provider %s", callerInfo)
    }

    // Add model parameter
    if err := writer.WriteField("model", model); err != nil {
        return nil, fmt.Errorf("failed to write model field %s: %w", callerInfo, err)
    }

    // Add response format
    responseFormat := "json"
    if req.OutputFormat == "verbose_json" {
        responseFormat = "verbose_json"
    }
    if err := writer.WriteField("response_format", responseFormat); err != nil {
        return nil, fmt.Errorf("failed to write response_format field %s: %w", callerInfo, err)
    }

    // Add language if specified
    if req.Language != "" && req.Language != "auto" {
        if err := writer.WriteField("language", req.Language); err != nil {
            return nil, fmt.Errorf("failed to write language field %s: %w", callerInfo, err)
        }
    }

    // Add audio file
    part, err := writer.CreateFormFile("file", req.FileName)
    if err != nil {
        return nil, fmt.Errorf("failed to create form file %s: %w", callerInfo, err)
    }
    if _, err := io.Copy(part, bytes.NewReader(req.AudioData)); err != nil {
        return nil, fmt.Errorf("failed to write audio data to form %s: %w", callerInfo, err)
    }
    if err := writer.Close(); err != nil {
        return nil, fmt.Errorf("failed to close multipart writer %s: %w", callerInfo, err)
    }

    httpReq, err := http.NewRequest("POST", ts.WhisperURL, body)
    if err != nil {
        return nil, fmt.Errorf("failed to create HTTP request %s: %w", callerInfo, err)
    }
    httpReq.Header.Set("Content-Type", writer.FormDataContentType())

    if ts.APIKey == "" {
        return nil, fmt.Errorf("API key is required for Groq provider %s", callerInfo)
    }
    httpReq.Header.Set("Authorization", "Bearer "+ts.APIKey)

    client := &http.Client{}
    resp, err := client.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("failed to send request to Groq %s: %w", callerInfo, err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("Groq service returned error %s: Status %d - %s", 
            callerInfo, resp.StatusCode, string(body))
    }

    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read response body %s: %w", callerInfo, err)
    }

    // Parse Groq response
    var groqResponse struct {
        Text     string `json:"text"`
        Language string `json:"language"`
    }
    if err := json.Unmarshal(respBody, &groqResponse); err != nil {
        return nil, fmt.Errorf("failed to parse Groq response %s: %w", callerInfo, err)
    }

    // Convert to our standard response format
    result := &TranscribeResponse{
        Text:     groqResponse.Text,
        Language: groqResponse.Language,
    }

    return result, nil
}

// GetWhisperURL returns the Whisper URL with default values if not set
func GetWhisperURL() (string, error) {
    whisperURL := os.Getenv("WHISPER_URL")
    if whisperURL == "" {
        return "", fmt.Errorf("WHISPER_URL environment variable is not set")
    }
    return whisperURL, nil
}
