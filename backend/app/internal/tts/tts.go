package tts

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
)

// TTSService handles text-to-speech conversion
type TTSService struct {
    BaseURL        string
    APIKey         string
    Model          string
    Voice          string
    ResponseFormat string
    Speed          float64
}

// NewTTSService creates a new TTS service
func NewTTSService() (*TTSService, error) {
    baseURL := os.Getenv("TTS_BASE_URL")
    if baseURL == "" {
        return nil, fmt.Errorf("TTS_BASE_URL environment variable is not set")
    }

    apiKey := os.Getenv("TTS_API_KEY")
    model := os.Getenv("TTS_MODEL")
    voice := os.Getenv("TTS_VOICE")
    responseFormat := os.Getenv("TTS_RESPONSE_FORMAT")

    return &TTSService{
        BaseURL:        baseURL,
        APIKey:         apiKey,
        Model:          model,
        Voice:          voice,
        ResponseFormat: responseFormat,
    }, nil
}

// TTSRequest represents a TTS request
type TTSRequest struct {
    Text           string  `json:"text"`
    Model          string  `json:"model"`
    Voice          string  `json:"voice,omitempty"`
    ResponseFormat string  `json:"response_format,omitempty"`
    Speed          float64 `json:"speed,omitempty"`
}

// TTSResponse represents a TTS response
type TTSResponse struct {
    AudioData []byte
    Error     string
}

// ConvertTextToSpeech converts text to speech
func (ts *TTSService) ConvertTextToSpeech(req *TTSRequest) (*TTSResponse, error) {
    url := fmt.Sprintf("%s/v1/audio/speech", ts.BaseURL)

    // Use the service's default values if not provided in the request
    if req.Model == "" {
        req.Model = ts.Model
    }
    if req.Voice == "" {
        req.Voice = ts.Voice
    }
    if req.ResponseFormat == "" {
        req.ResponseFormat = ts.ResponseFormat
    }

    jsonData, err := json.Marshal(req)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request: %w", err)
    }

    httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, fmt.Errorf("failed to create HTTP request: %w", err)
    }

    httpReq.Header.Set("Content-Type", "application/json")
    if ts.APIKey != "" {
        httpReq.Header.Set("Authorization", "Bearer "+ts.APIKey)
    }

    client := &http.Client{}
    resp, err := client.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("failed to send request to TTS service: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return &TTSResponse{
            Error: fmt.Sprintf("TTS service returned non-OK status: %s - %s", resp.Status, string(body)),
        }, nil
    }

    audioData, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read audio data: %w", err)
    }

    return &TTSResponse{
        AudioData: audioData,
    }, nil
}
