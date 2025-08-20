package handlers

import (
    "context"
    "encoding/json"
    "log"
    "net/http"
    "os"

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

        req := &whisper.TranscribeRequest{
            AudioData:    audioData,
            FileName:     fileName,
            Language:     "en",
            Task:         "transcribe",
            OutputFormat: "json",
            ShouldEncode: true,
        }

        result, err := ts.SendToWhisper(req)
        if err != nil {
            http.Error(w, "Transcription failed: "+err.Error(), http.StatusInternalServerError)
            return
        }

        log.Printf("Transcribed text: %s", result.Text)

        // Initialize OpenAI client
        client := openai.NewClient(
            option.WithBaseURL(os.Getenv("OPENAI_BASE_URL")),
            option.WithAPIKey(os.Getenv("OPENAI_KEY")),
        )

        // Send transcribed text to OPENAI compatable endpoint
        chatCompletion, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
            Messages: []openai.ChatCompletionMessageParamUnion{
                openai.UserMessage(result.Text),
            },
            Model: openai.ChatModel(os.Getenv("OPENAI_MODEL")),
        })
        if err != nil {
            log.Printf("OPENAI request failed: %v", err)
            http.Error(w, "OPENAI request failed: "+err.Error(), http.StatusInternalServerError)
            return
        }

        // Log Ollama response
        ollamaResponse := chatCompletion.Choices[0].Message.Content
        log.Printf("OPENAI response: %s", ollamaResponse)

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]string{
            "status": "success",
            "transcribed_text": result.Text,
            "ollama_response": ollamaResponse,
        })
    }
}
