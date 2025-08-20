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
    "github.com/jackc/pgx/v5"
)

// ConversationTurn represents a single turn in the conversation
type ConversationTurn struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

func APIConversationHandler(ts *whisper.TranscribeService) func(http.ResponseWriter, *http.Request, *pgx.Conn) {
    return func(w http.ResponseWriter, r *http.Request, conn *pgx.Conn) {
        // Parse the multipart form
        if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB
            http.Error(w, "Unable to parse form", http.StatusBadRequest)
            return
        }

        // Get the history from the form
        historyJSON := r.FormValue("history")
        var history []ConversationTurn
        if historyJSON != "" {
            if err := json.Unmarshal([]byte(historyJSON), &history); err != nil {
                http.Error(w, "Invalid history format", http.StatusBadRequest)
                return
            }
        }

        // Add hello message as first turn if history is empty
        if len(history) == 0 {
            history = []ConversationTurn{
                {
                    Role:    "assistant",
                    Content: "Hello! What would you like to talk about today?",
                },
            }
        }

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

        // Build messages for LLM with history
        messages := []openai.ChatCompletionMessageParamUnion{
            openai.SystemMessage("You are a helpful language learning assistant. Keep responses concise and engaging not longer than 2 sentences. Do not use emojis or markdown in your responses."),
        }
        
        // Add conversation history
        for _, turn := range history {
            if turn.Role == "user" {
                messages = append(messages, openai.UserMessage(turn.Content))
            } else if turn.Role == "assistant" {
                messages = append(messages, openai.AssistantMessage(turn.Content))
            }
        }
        
        // Add current user message
        messages = append(messages, openai.UserMessage(result.Text))

        // Send transcribed text to LLM
        chatCompletion, err := llmClient.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
            Messages: messages,
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

        // Create TTS request
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

        // Encode audio to base64 for JSON response
        audioBase64 := base64.StdEncoding.EncodeToString(audioBytes)

        // Update history with new turns
        history = append(history, ConversationTurn{Role: "user", Content: result.Text})
        history = append(history, ConversationTurn{Role: "assistant", Content: llmResponse})

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]interface{}{
            "status":           "success",
            "transcribed_text": result.Text,
            "llm_response":     llmResponse,
            "audio_base64":     audioBase64,
            "history":          history,
        })
    }
}
