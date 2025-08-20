package handlers

import (
    "bytes"
    "context"
    "encoding/base64"
    "encoding/json"
    "io"
    "log"
    "net/http"
    "os"
    "strings"

    "github.com/gchalakovmmi/PulpuWEB/whisper"
    "github.com/openai/openai-go/v2"
    "github.com/openai/openai-go/v2/option"
)

func APIConversationHandler(ts *whisper.TranscribeService) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        audioData, fileName, err := whisper.ParseAudioFromRequest(r)
        if err != nil {
            http.Error(w, "Unable to process audio: "+err.Error(), http.StatusBadRequest)
            return
        }

        whisperReq := &whisper.TranscribeRequest{
            AudioData:    audioData,
            FileName:     fileName,
            Language:     "en",
            Task:         "transcribe",
            OutputFormat: "json",
            ShouldEncode: true,
        }

        result, err := ts.SendToWhisper(whisperReq)
        if err != nil {
            http.Error(w, "Transcription failed: "+err.Error(), http.StatusInternalServerError)
            return
        }

        log.Printf("Transcribed text: %s", result.Text)

        // Initialize OpenAI client for LLM
        llmClient := openai.NewClient(
            option.WithBaseURL(os.Getenv("OPENAI_BASE_URL")),
            option.WithAPIKey(os.Getenv("OPENAI_KEY")),
        )

        // Send transcribed text to LLM
        chatCompletion, err := llmClient.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
            Messages: []openai.ChatCompletionMessageParamUnion{
                openai.UserMessage(result.Text),
            },
            Model: openai.ChatModel(os.Getenv("OPENAI_MODEL")),
        })
        if err != nil {
            log.Printf("LLM request failed: %v", err)
            http.Error(w, "LLM request failed: "+err.Error(), http.StatusInternalServerError)
            return
        }

        // Log LLM response
        llmResponse := chatCompletion.Choices[0].Message.Content
        log.Printf("LLM response: %s", llmResponse)

        // Initialize HTTP client for TTS (KittenTTS)
        ttsURL := os.Getenv("OPENAI_TTS_URL") + "/v1/audio/speech"
        
        // Create TTS request - using exact parameters from your working curl command
        ttsRequest := map[string]interface{}{
            "model":           "kitten-tts",
            "input":           llmResponse,
            "voice":           "expr-voice-4-f",
            "response_format": "mp3",
            "speed":           0.9,
        }

        jsonData, err := json.Marshal(ttsRequest)
        if err != nil {
            log.Printf("Failed to marshal TTS request: %v", err)
            http.Error(w, "Failed to prepare TTS request", http.StatusInternalServerError)
            return
        }

        log.Printf("Sending TTS request to: %s", ttsURL)
        log.Printf("TTS request body: %s", string(jsonData))

        // Create HTTP request
        ttsReq, err := http.NewRequest("POST", ttsURL, bytes.NewBuffer(jsonData))
        if err != nil {
            log.Printf("Failed to create TTS request: %v", err)
            http.Error(w, "Failed to create TTS request", http.StatusInternalServerError)
            return
        }

        ttsReq.Header.Set("Content-Type", "application/json")

        // Send request
        client := &http.Client{}
        resp, err := client.Do(ttsReq)
        if err != nil {
            log.Printf("TTS request failed: %v", err)
            http.Error(w, "TTS request failed", http.StatusInternalServerError)
            return
        }
        defer resp.Body.Close()

        // Check if the response is successful
        if resp.StatusCode != http.StatusOK {
            body, _ := io.ReadAll(resp.Body)
            log.Printf("TTS server returned error: Status %d, Body: %s", resp.StatusCode, string(body))
            
            // Check if it's a JSON error response
            if strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
                var errorResponse map[string]interface{}
                if json.Unmarshal(body, &errorResponse) == nil {
                    if detail, ok := errorResponse["detail"].(string); ok {
                        http.Error(w, "TTS server error: "+detail, http.StatusInternalServerError)
                        return
                    }
                }
            }
            
            http.Error(w, "TTS server error", http.StatusInternalServerError)
            return
        }

        // Read the audio data
        audioBytes, err := io.ReadAll(resp.Body)
        if err != nil {
            log.Printf("Failed to read TTS audio: %v", err)
            http.Error(w, "Failed to process TTS audio", http.StatusInternalServerError)
            return
        }

        // Check if we actually got audio data
        contentType := resp.Header.Get("Content-Type")
        if !strings.Contains(contentType, "audio/") && !strings.Contains(contentType, "application/octet-stream") {
            log.Printf("Unexpected content type from TTS server: %s", contentType)
            http.Error(w, "TTS server returned unexpected content type: "+contentType, http.StatusInternalServerError)
            return
        }

        // Check if the audio data is valid (MP3 files typically start with ID3 tag or FF FB)
        if len(audioBytes) < 2 || (audioBytes[0] != 0xFF && audioBytes[0] != 'I') {
            log.Printf("Invalid audio data received from TTS server")
            http.Error(w, "Invalid audio data received from TTS server", http.StatusInternalServerError)
            return
        }

        // Encode audio to base64 for JSON response
        audioBase64 := base64.StdEncoding.EncodeToString(audioBytes)

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]string{
            "status":           "success",
            "transcribed_text": result.Text,
            "llm_response":     llmResponse,
            "audio_base64":     audioBase64,
        })
    }
}
