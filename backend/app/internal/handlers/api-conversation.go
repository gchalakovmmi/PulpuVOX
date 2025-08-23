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
    "regexp"
    "strings"
    "sync"
    "github.com/gchalakovmmi/PulpuWEB/whisper"
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

// limitResponseLength limits the response to a maximum number of sentences and characters
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

        // Initialize HTTP client for TTS (KittenTTS)
        ttsURL := os.Getenv("OPENAI_TTS_URL") + "/v1/audio/speech"

        // Create TTS request with limited response
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
            sendJSONError("Failed to prepare TTS request", http.StatusInternalServerError)
            return
        }

        // Create HTTP request
        ttsReq, err := http.NewRequest("POST", ttsURL, bytes.NewBuffer(jsonData))
        if err != nil {
            log.Printf("Failed to create TTS request: %v", err)
            sendJSONError("Failed to create TTS request", http.StatusInternalServerError)
            return
        }
        ttsReq.Header.Set("Content-Type", "application/json")

        // Send request
        client := &http.Client{}
        resp, err := client.Do(ttsReq)
        if err != nil {
            log.Printf("TTS request failed: %v", err)
            sendJSONError("TTS request failed", http.StatusInternalServerError)
            return
        }
        defer resp.Body.Close()

        // Check if the response is successful
        if resp.StatusCode != http.StatusOK {
            body, _ := io.ReadAll(resp.Body)
            log.Printf("TTS server returned error: Status %d, Body: %s", resp.StatusCode, string(body))
            
            // Even if TTS fails, we can still return the text response
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

        // Read the audio data
        audioBytes, err := io.ReadAll(resp.Body)
        if err != nil {
            log.Printf("Failed to read TTS audio: %v", err)
            sendJSONError("Failed to process TTS audio", http.StatusInternalServerError)
            return
        }

        // Check if we actually got audio data
        contentType := resp.Header.Get("Content-Type")
        if !strings.Contains(contentType, "audio/") && !strings.Contains(contentType, "application/octet-stream") {
            log.Printf("Unexpected content type from TTS server: %s", contentType)
            sendJSONError("TTS server returned unexpected content type", http.StatusInternalServerError)
            return
        }

        // Encode audio to base64 for JSON response
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
