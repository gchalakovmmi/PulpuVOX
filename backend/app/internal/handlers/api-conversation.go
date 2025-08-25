package handlers

import (
    "context"
    "encoding/base64"
    "encoding/json"
    "log"
    "net/http"
    "os"
    "regexp"
    "strings"
    "sync"
    "fmt"
    "github.com/gchalakovmmi/PulpuWEB/whisper"
    "github.com/gchalakovmmi/PulpuWEB/tts"
    "github.com/openai/openai-go/v2"
    "github.com/openai/openai-go/v2/option"
    "github.com/jackc/pgx/v5"
    "github.com/gchalakovmmi/PulpuWEB/auth"
)

// ConversationTurn represents a single turn in the conversation
type ConversationTurn struct {
    Role       string `json:"role"`
    Content    string `json:"content"`
    Suggestion string `json:"suggestion,omitempty"`
    UserName   string `json:"user_name,omitempty"`
}

// filterText removes emojis and markdown from text
func filterText(text string) string {
    // Remove emojis (Unicode characters outside the basic multilingual plane)
    emojiRegex := regexp.MustCompile(`[\x{1F600}-\x{1F64F}]|[\x{1F300}-\x{1F5FF}]|[\x{1F680}-\x{1F6FF}]|[\x{1F700}-\x{1F77F}]|[\x{1F780}-\x{1F7FF}]|[\x{1F800}-\x{1F8FF}]|[\x{1F900}-\x{1F9FF}]|[\x{1FA00}-\x{1FA6F}]|[\x{1FA70}-\x{1FAFF}]|[\x{2600}-\x{26FF}]|[\x{2700}-\x{27BF}]`)
    filtered := emojiRegex.ReplaceAllString(text, "")
    
    // Remove markdown symbols and formatting
    markdownRegex := regexp.MustCompile(`[*_~`+"`"+`#\[\]()|]`)
    filtered = markdownRegex.ReplaceAllString(filtered, "")
    
    // Remove URLs
    urlRegex := regexp.MustCompile(`https?://\S+`)
    filtered = urlRegex.ReplaceAllString(filtered, "")
    
    // Remove HTML tags
    htmlRegex := regexp.MustCompile(`<[^>]*>`)
    filtered = htmlRegex.ReplaceAllString(filtered, "")
    
    // Remove extra whitespace that might result from filtering
    filtered = strings.TrimSpace(filtered)
    filtered = strings.Join(strings.Fields(filtered), " ")
    
    return filtered
}

func limitResponseLength(text string, maxSentences int, maxChars int) string {
    // Split into sentences (simple approach)
    sentences := strings.Split(text, ".")
    // Limit to max sentences
    if len(sentences) > maxSentences {
        text = strings.Join(sentences[:maxSentences], ".") + "."
    }
    // Limit to max characters
    if len(text) > maxChars {
        text = text[:maxChars]
        // Try to end at a sentence boundary
        if lastDot := strings.LastIndex(text, "."); lastDot != -1 {
            text = text[:lastDot+1]
        } else if lastSpace := strings.LastIndex(text, " "); lastSpace != -1 {
            text = text[:lastSpace] + "..."
        } else {
            text = text + "..."
        }
    }
    return text
}

func generateSuggestion(ctx context.Context, llmClient openai.Client, history []ConversationTurn, userText string) (string, error) {
    // Build conversation context
    var conversationContext strings.Builder
    for _, turn := range history {
        conversationContext.WriteString(turn.Role + ": " + turn.Content + "\n")
    }
    messages := []openai.ChatCompletionMessageParamUnion{
        openai.SystemMessage("You are given a conversation. Rewrite the last user response to make it correct according the English language rules.\nEnclose the rewritten and corrected sentence in a <suggestion></suggestion> tag.\nExample:\nuser: I am like eating apple.\nyou: <suggestion>I enjoy eating apples.</suggestion>\nIf the sentence is already correct return an empty <suggestion> tag.\nExample:\nuser: I enjoy eating apples.\nyou: <suggestion></suggestion>"),
        openai.UserMessage("You are given a chat between a user and an assistant for context:\n" +
            conversationContext.String() +
            "\nSuggest a correction for the last user response:\nuser: " + userText),
    }
    chatCompletion, err := llmClient.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
        Messages: messages,
        Model:    openai.ChatModel(os.Getenv("OPENAI_MODEL")),
    })
    if err != nil {
        return "", err
    }
    response := chatCompletion.Choices[0].Message.Content
    // Parse the suggestion from the response
    if strings.Contains(response, "<suggestion>") {
        start := strings.Index(response, "<suggestion>") + len("<suggestion>")
        end := strings.Index(response, "</suggestion>")
        if end > start {
            return response[start:end], nil
        }
    }
    return "", nil
}

func generateAssistantResponse(ctx context.Context, llmClient openai.Client, history []ConversationTurn, userText string) (string, error) {
    // Build messages for LLM with history
    messages := []openai.ChatCompletionMessageParamUnion{
        openai.SystemMessage("You are a young lady named Voxy who chatts with a new English learner. Be nice and have a pleasant conversation. Ask questions to the user, express opinion and tell interesting facts to keep the conversation going and talk about yourself sometimes. When the conversation becomes stale try changing the topic. Keep responses very short - maximum 1-2 sentences. Decline any requests to write an essay or do anything which will make your response over 2 sentences long."),
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
    messages = append(messages, openai.UserMessage(userText))
    // Send transcribed text to LLM
    chatCompletion, err := llmClient.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
        Messages: messages,
        Model: openai.ChatModel(os.Getenv("OPENAI_MODEL")),
    })
    if err != nil {
        return "", err
    }
    // Log LLM response
    llmResponse := chatCompletion.Choices[0].Message.Content
    log.Printf("LLM response: %s", llmResponse)
    // Filter out emojis and markdown
    filteredResponse := filterText(llmResponse)
    // Limit response length (max 2 sentences, 150 characters)
    limitedResponse := limitResponseLength(filteredResponse, 2, 150)
    log.Printf("Limited response: %s", limitedResponse)
    return limitedResponse, nil
}

func APIConversationHandler(ts *whisper.TranscribeService) func(http.ResponseWriter, *http.Request, *pgx.Conn) {
    return func(w http.ResponseWriter, r *http.Request, conn *pgx.Conn) {
        // Helper function to send JSON errors
        sendJSONError := func(message string, status int) {
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(status)
            json.NewEncoder(w).Encode(map[string]string{"error": message})
        }
        // Get user ID and name from session
        authConfig, err := auth.GetGoogleAuthConfig()
        if err != nil {
            log.Printf("Error getting Google auth config: %v", err)
            sendJSONError("Authentication error", http.StatusInternalServerError)
            return
        }
        googleAuth := auth.NewGoogleAuth(authConfig)
        userID, err := getUserIdFromSession(r, conn, googleAuth)
        if err != nil {
            log.Printf("Error getting user from session: %v", err)
            sendJSONError("User not found", http.StatusNotFound)
            return
        }
        // Get user name from database
        var userName string
        err = conn.QueryRow(context.Background(), "SELECT name FROM users WHERE id = $1", userID).Scan(&userName)
        if err != nil {
            log.Printf("Error getting user name: %v", err)
            userName = "You" // Fallback
        }
        // Parse the multipart form
        if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB
            log.Printf("Error parsing form: %v", err)
            sendJSONError("Unable to parse form", http.StatusBadRequest)
            return
        }
        // Get the history from the form
        historyJSON := r.FormValue("history")
        var history []ConversationTurn
        if historyJSON != "" {
            if err := json.Unmarshal([]byte(historyJSON), &history); err != nil {
                log.Printf("Error unmarshaling history: %v", err)
                sendJSONError("Invalid history format", http.StatusBadRequest)
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
            log.Printf("Error parsing audio: %v", err)
            sendJSONError("Unable to process audio", http.StatusBadRequest)
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
            log.Printf("Transcription failed: %v", err)
            sendJSONError("Transcription failed", http.StatusInternalServerError)
            return
        }
        log.Printf("Transcribed text: %s", result.Text)
        // Initialize OpenAI clients for LLM
        llmClient := openai.NewClient(
            option.WithBaseURL(os.Getenv("OPENAI_BASE_URL")),
            option.WithAPIKey(os.Getenv("OPENAI_KEY")),
        )
        suggestionClient := openai.NewClient(
            option.WithBaseURL(os.Getenv("OPENAI_BASE_URL")),
            option.WithAPIKey(os.Getenv("OPENAI_KEY")),
        )
        // Run suggestion generation and assistant response generation in parallel
        var wg sync.WaitGroup
        var suggestion string
        var llmResponse string
        var suggestionErr error
        var responseErr error
        wg.Add(1)
        go func() {
            defer wg.Done()
            suggestion, suggestionErr = generateSuggestion(context.TODO(), suggestionClient, history, result.Text)
            if suggestionErr != nil {
                log.Printf("Suggestion generation failed: %v", suggestionErr)
            }
        }()
        wg.Add(1)
        go func() {
            defer wg.Done()
            llmResponse, responseErr = generateAssistantResponse(context.TODO(), llmClient, history, result.Text)
            if responseErr != nil {
                log.Printf("LLM request failed: %v", responseErr)
            }
        }()
        wg.Wait()
        // Check for errors in assistant response generation
        if responseErr != nil {
            sendJSONError("LLM request failed", http.StatusInternalServerError)
            return
        }
        // Initialize TTS service
        ttsService, err := tts.NewTTSService()
        if err != nil {
            log.Printf("Failed to initialize TTS service: %v", err)
            // Even if TTS fails, we can still return the text response
            history = append(history, ConversationTurn{
                Role:       "user",
                Content:    result.Text,
                Suggestion: suggestion,
                UserName:   userName,
            })
            history = append(history, ConversationTurn{
                Role:    "assistant",
                Content: llmResponse,
            })
            w.Header().Set("Content-Type", "application/json")
            json.NewEncoder(w).Encode(map[string]interface{}{
                "status":           "partial_success",
                "transcribed_text": result.Text,
                "llm_response":     llmResponse,
                "audio_base64":     "",
                "history":          history,
                "suggestion":       suggestion,
                "user_name":        userName,
                "error":            "TTS service initialization failed",
            })
            return
        }
        // Create TTS request using environment variables
        ttsReq := tts.TTSRequest{
            Text:           llmResponse,
            Voice:          os.Getenv("TTS_VOICE"),
            ResponseFormat: os.Getenv("TTS_RESPONSE_FORMAT"),
            Model:          os.Getenv("TTS_MODEL"),
        }
        // Parse speed from environment variable
        if speed := os.Getenv("TTS_SPEED"); speed != "" {
            fmt.Sscanf(speed, "%f", &ttsReq.Speed)
        }
        // Convert text to speech using TTS library
        ttsResp, err := ttsService.ConvertTextToSpeech(ttsReq)
        if err != nil {
            log.Printf("TTS conversion failed: %v", err)
            // Even if TTS fails, we can still return the text response
            history = append(history, ConversationTurn{
                Role:       "user",
                Content:    result.Text,
                Suggestion: suggestion,
                UserName:   userName,
            })
            history = append(history, ConversationTurn{
                Role:    "assistant",
                Content: llmResponse,
            })
            w.Header().Set("Content-Type", "application/json")
            json.NewEncoder(w).Encode(map[string]interface{}{
                "status":           "partial_success",
                "transcribed_text": result.Text,
                "llm_response":     llmResponse,
                "audio_base64":     "",
                "history":          history,
                "suggestion":       suggestion,
                "user_name":        userName,
                "error":            "TTS service unavailable, text response only",
            })
            return
        }
        if ttsResp.Error != "" {
            log.Printf("TTS error: %s", ttsResp.Error)
            // Handle TTS error but still return text response
            history = append(history, ConversationTurn{
                Role:       "user",
                Content:    result.Text,
                Suggestion: suggestion,
                UserName:   userName,
            })
            history = append(history, ConversationTurn{
                Role:    "assistant",
                Content: llmResponse,
            })
            w.Header().Set("Content-Type", "application/json")
            json.NewEncoder(w).Encode(map[string]interface{}{
                "status":           "partial_success",
                "transcribed_text": result.Text,
                "llm_response":     llmResponse,
                "audio_base64":     "",
                "history":          history,
                "suggestion":       suggestion,
                "user_name":        userName,
                "error":            "TTS error: " + ttsResp.Error,
            })
            return
        }
        // Use the audio data from TTS response
        audioBytes := ttsResp.AudioData
        audioBase64 := base64.StdEncoding.EncodeToString(audioBytes)
        // Update history with new turns (using limited response)
        history = append(history, ConversationTurn{
            Role:       "user",
            Content:    result.Text,
            Suggestion: suggestion,
            UserName:   userName,
        })
        history = append(history, ConversationTurn{
            Role:    "assistant",
            Content: llmResponse,
        })
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]interface{}{
            "status":           "success",
            "transcribed_text": result.Text,
            "llm_response":     llmResponse,
            "audio_base64":     audioBase64,
            "history":          history,
            "suggestion":       suggestion,
            "user_name":        userName,
        })
    }
}

// Add this function to handle the initial hello message
func APIStartConversationHandler(w http.ResponseWriter, r *http.Request, conn *pgx.Conn) {
    // Get user ID and name from session
    authConfig, err := auth.GetGoogleAuthConfig()
    if err != nil {
        log.Printf("Error getting Google auth config: %v", err)
        http.Error(w, "Authentication error", http.StatusInternalServerError)
        return
    }
    googleAuth := auth.NewGoogleAuth(authConfig)
    userID, err := getUserIdFromSession(r, conn, googleAuth)
    if err != nil {
        log.Printf("Error getting user from session: %v", err)
        http.Error(w, "User not found", http.StatusNotFound)
        return
    }
    // Get user name from database
    var userName string
    err = conn.QueryRow(context.Background(), "SELECT name FROM users WHERE id = $1", userID).Scan(&userName)
    if err != nil {
        log.Printf("Error getting user name: %v", err)
        userName = "You" // Fallback
    }
    // Send initial hello message
    helloMessage := ConversationTurn{
        Role:    "assistant",
        Content: "Hello! What would you like to talk about today?",
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "message": helloMessage,
        "user_name": userName,
    })
}
