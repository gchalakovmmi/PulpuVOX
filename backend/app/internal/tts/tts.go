package tts

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "runtime"
    "strconv"
)

// TTSProvider represents the available TTS providers
type TTSProvider string

const (
    ProviderKittenTTS TTSProvider = "kittentts"
    ProviderGroq      TTSProvider = "groq"
)

// TTSService handles text-to-speech conversion
type TTSService struct {
    BaseURL        string
    APIKey         string
    Model          string
    Voice          string
    ResponseFormat string
    Speed          float64
    Provider       TTSProvider
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
    
    // Parse speed from environment
    speed := 1.0
    if speedStr := os.Getenv("TTS_SPEED"); speedStr != "" {
        if parsedSpeed, err := strconv.ParseFloat(speedStr, 64); err == nil {
            speed = parsedSpeed
        }
    }
    
    // Determine provider from environment variable
    providerStr := os.Getenv("TTS_PROVIDER")
    var provider TTSProvider
    switch providerStr {
    case "groq":
        provider = ProviderGroq
    case "kittentts":
        provider = ProviderKittenTTS
    default:
        provider = ProviderKittenTTS // Default to KittenTTS
    }

    return &TTSService{
        BaseURL:        baseURL,
        APIKey:         apiKey,
        Model:          model,
        Voice:          voice,
        ResponseFormat: responseFormat,
        Speed:          speed,
        Provider:       provider,
    }, nil
}

// TTSRequest represents a TTS request
type TTSRequest struct {
    Text           string  `json:"input"`
    Model          string  `json:"model,omitempty"`
    Voice          string  `json:"voice,omitempty"`
    ResponseFormat string  `json:"response_format,omitempty"`
    Speed          float64 `json:"speed,omitempty"`
}

// TTSResponse represents a TTS response
type TTSResponse struct {
    AudioData []byte
    Error     string
}

// getCallerInfo returns file and line information for error reporting
func getCallerInfo() string {
    pc, file, line, ok := runtime.Caller(2) // Skip 2 frames to get the actual caller
    if ok {
        return fmt.Sprintf("at %s:%d (%s)", file, line, runtime.FuncForPC(pc).Name())
    }
    return "at unknown location"
}

// ConvertTextToSpeech converts text to speech
func (ts *TTSService) ConvertTextToSpeech(req *TTSRequest) (*TTSResponse, error) {
    callerInfo := getCallerInfo()
    
    switch ts.Provider {
    case ProviderGroq:
        return ts.convertWithGroq(req, callerInfo)
    case ProviderKittenTTS:
        fallthrough
    default:
        return ts.convertWithKittenTTS(req, callerInfo)
    }
}

// convertWithKittenTTS converts text to speech using KittenTTS
func (ts *TTSService) convertWithKittenTTS(req *TTSRequest, callerInfo string) (*TTSResponse, error) {
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
    if req.Speed == 0 {
        req.Speed = ts.Speed
    }

    // Create TTS request - ensure this matches exactly what KittenTTS expects
    ttsRequest := map[string]interface{}{
        "model":           req.Model,
        "input":           req.Text,
        "voice":           req.Voice,
        "response_format": req.ResponseFormat,
        "speed":           req.Speed,
    }
    
    jsonData, err := json.Marshal(ttsRequest)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal TTS request %s: %w", callerInfo, err)
    }

    // Create HTTP request
    httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, fmt.Errorf("failed to create TTS request %s: %w", callerInfo, err)
    }
    httpReq.Header.Set("Content-Type", "application/json")
    
    // KittenTTS might not require authentication, but include it if provided
    if ts.APIKey != "" {
        httpReq.Header.Set("Authorization", "Bearer "+ts.APIKey)
    }

    // Send request
    client := &http.Client{}
    resp, err := client.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("TTS request failed %s: %w", callerInfo, err)
    }
    defer resp.Body.Close()

    // Check if the response is successful
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return &TTSResponse{
            Error: fmt.Sprintf("TTS server returned error %s: Status %d, Body: %s", callerInfo, resp.StatusCode, string(body)),
        }, nil
    }

    // Read the audio data
    audioBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read TTS audio %s: %w", callerInfo, err)
    }

    return &TTSResponse{
        AudioData: audioBytes,
    }, nil
}

// convertWithGroq converts text to speech using Groq API
func (ts *TTSService) convertWithGroq(req *TTSRequest, callerInfo string) (*TTSResponse, error) {
    // Groq uses a different endpoint structure
    url := fmt.Sprintf("%s/openai/v1/audio/speech", ts.BaseURL)
    
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

    // Create TTS request for Groq
    ttsRequest := map[string]interface{}{
        "model":    req.Model,
        "input":    req.Text,
        "voice":    req.Voice,
        "response_format": req.ResponseFormat,
    }
    
    jsonData, err := json.Marshal(ttsRequest)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal TTS request %s: %w", callerInfo, err)
    }

    // Create HTTP request
    httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, fmt.Errorf("failed to create TTS request %s: %w", callerInfo, err)
    }
    httpReq.Header.Set("Content-Type", "application/json")
    if ts.APIKey != "" {
        httpReq.Header.Set("Authorization", "Bearer "+ts.APIKey)
    }

    // Send request
    client := &http.Client{}
    resp, err := client.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("TTS request failed %s: %w", callerInfo, err)
    }
    defer resp.Body.Close()

    // Check if the response is successful
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return &TTSResponse{
            Error: fmt.Sprintf("TTS server returned error %s: Status %d, Body: %s", callerInfo, resp.StatusCode, string(body)),
        }, nil
    }

    // Read the audio data
    audioBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read TTS audio %s: %w", callerInfo, err)
    }

    return &TTSResponse{
        AudioData: audioBytes,
    }, nil
}
