package conversation

import (
    "encoding/base64"
    "encoding/json"
    "io"
    "log"
    "net/http"
    "regexp"
    "context"
    "strings"
    "sync"
    "PulpuVOX/internal/openai"
    "PulpuVOX/internal/tts"
    "PulpuVOX/internal/whisper"
    "github.com/jackc/pgx/v5"
    "github.com/markbates/goth"
)

// ConversationTurn represents a single turn in the conversation
type ConversationTurn struct {
    Role string `json:"role"`
    Content string `json:"content"`
    Suggestion string `json:"suggestion,omitempty"`
    UserName string `json:"user_name,omitempty"`
}

// filterText removes emojis and markdown from text
func filterText(text string) string {
    // Remove emojis (Unicode characters outside the basic multilingual plane)
    emojiRegex := regexp.MustCompile(`[\x{1F600}-\x{1F64F}]|[\x{1F300}-\x{1F5FF}]|[\x{1F680}-\x{1F6FF}]|[\x{1F700}-\x{1F77F}]|[\x{1F780}-\x{1F7FF}]|[\x{1F800}-\x{1F8FF}]|[\x{1F900}-\x{1F9FF}]|[\x{1FA00}-\x{1FA6F}]|[\x{1FA70}-\x{1FAFF}]|[\x{2600}-\x{26FF}]|[\x{2700}-\x{27BF}]`)
    filtered := emojiRegex.ReplaceAllString(text, "")
    
    // Remove markdown symbols and formatting
    markdownRegex := regexp.MustCompile(`[*_~` + "`" + `+#\[\]()|]`)
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

func limitResponseLength(text string, maxSentences int) string {
    // Split into sentences using multiple sentence terminators
    sentenceEnders := regexp.MustCompile(`([.!?]+\s*)`)
    parts := sentenceEnders.Split(text, -1)
    separators := sentenceEnders.FindAllString(text, -1)
    
    // Reconstruct sentences with their terminators
    var sentences []string
    for i, part := range parts {
        if i < len(separators) {
            part += separators[i]
        }
        part = strings.TrimSpace(part)
        if part != "" {
            sentences = append(sentences, part)
        }
    }
    
    // Limit to max sentences
    if len(sentences) > maxSentences {
        return strings.Join(sentences[:maxSentences], " ")
    }
    
    return strings.Join(sentences, " ")
}

func generateSuggestion(ctx context.Context, llmClient *openai.Client, history []ConversationTurn, userText string) (string, error) {
    // Build conversation context
    var conversationContext strings.Builder
    for _, turn := range history {
        conversationContext.WriteString(turn.Role + ": " + turn.Content + "\n")
    }
    
    messages := []openai.ChatCompletionMessage{
        {
            Role: "system",
            Content: "You are given a conversation. Rewrite the last user response to make it correct according the English language rules.\nEnclose the rewritten and corrected sentence in a <suggestion></suggestion> tag.\nExample:\nuser: I am like eating apple.\nyou: <suggestion>I enjoy eating apples.</suggestion>\nIf the sentence is already correct return an empty <suggestion> tag.\nExample:\nuser: I enjoy eating apples.\nyou: <suggestion></suggestion>",
        },
        {
            Role: "user",
            Content: "You are given a chat between a user and an assistant for context:\n" +
                conversationContext.String() +
                "\nSuggest a correction for the last user response:\nuser: " + userText,
        },
    }
    
    chatCompletion, err := llmClient.CreateChatCompletion(&openai.ChatCompletionRequest{
        Messages: messages,
        Model: llmClient.Model,
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

func generateAssistantResponse(ctx context.Context, llmClient *openai.Client, history []ConversationTurn, userText string) (string, error) {
    // Build messages for LLM with history
    messages := []openai.ChatCompletionMessage{
        {
            Role: "system",
            Content: "You are a young lady named Voxy who chatts with a new English learner. Be nice and have a pleasant conversation. Ask questions to the user, express opinion and tell interesting facts to keep the conversation going and talk about yourself sometimes. When the conversation becomes stale try changing the topic. Keep responses very short - maximum 1-2 sentences. Decline any requests to write an essay or do anything which will make your response over 2 sentences long.",
        },
    }
    
    // Add conversation history
    for _, turn := range history {
        messages = append(messages, openai.ChatCompletionMessage{
            Role: turn.Role,
            Content: turn.Content,
        })
    }
    
    // Add current user message
    messages = append(messages, openai.ChatCompletionMessage{
        Role: "user",
        Content: userText,
    })
    
    // Send to LLM
    chatCompletion, err := llmClient.CreateChatCompletion(&openai.ChatCompletionRequest{
        Messages: messages,
        Model: llmClient.Model,
    })
    if err != nil {
        return "", err
    }
    
    // Log LLM response
    llmResponse := chatCompletion.Choices[0].Message.Content
    log.Printf("LLM response: %s", llmResponse)
    
    // Filter out emojis and markdown
    filteredResponse := filterText(llmResponse)
    
    // Limit response length (max 4 sentences)
    limitedResponse := limitResponseLength(filteredResponse, 4)
    log.Printf("Limited response: %s", limitedResponse)
    
    return limitedResponse, nil
}

// APIConversationHandler handles the conversation API endpoint
func APIConversationHandler(whisperService *whisper.TranscribeService, llmClient *openai.Client, ttsService *tts.TTSService) func(http.ResponseWriter, *http.Request, *pgx.Conn) {
    return func(w http.ResponseWriter, r *http.Request, conn *pgx.Conn) {
        // Helper function to send JSON errors
        sendJSONError := func(message string, status int) {
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(status)
            json.NewEncoder(w).Encode(map[string]string{"error": message})
        }
        
        // Get user from context
        user, ok := r.Context().Value("user").(*goth.User)
        if !ok || user == nil {
            sendJSONError("User not authenticated", http.StatusUnauthorized)
            return
        }
        
        // Get user name from database
        var userName string
        err := conn.QueryRow(r.Context(), "SELECT name FROM users WHERE provider = $1 AND id_by_provider = $2", user.Provider, user.UserID).Scan(&userName)
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
        
        // Get the audio file
        file, _, err := r.FormFile("audio")
        if err != nil {
            log.Printf("Error getting audio file: %v", err)
            sendJSONError("Unable to get audio file", http.StatusBadRequest)
            return
        }
        defer file.Close()
        
        audioData, err := io.ReadAll(file)
        if err != nil {
            log.Printf("Error reading audio data: %v", err)
            sendJSONError("Unable to read audio data", http.StatusBadRequest)
            return
        }
        
        // Transcribe audio
        whisperReq := &whisper.TranscribeRequest{
            AudioData: audioData,
            FileName: "recording.mp3",
            Language: "en",
            Task: "transcribe",
            OutputFormat: "json",
        }
        
        result, err := whisperService.SendToWhisper(whisperReq)
        if err != nil {
            log.Printf("Transcription failed: %v", err)
            sendJSONError("Transcription failed", http.StatusInternalServerError)
            return
        }
        
        log.Printf("Transcribed text: %s", result.Text)
        
        // Run suggestion generation and assistant response generation in parallel
        var wg sync.WaitGroup
        var suggestion string
        var llmResponse string
        var suggestionErr error
        var responseErr error
        
        wg.Add(1)
        go func() {
            defer wg.Done()
            suggestion, suggestionErr = generateSuggestion(r.Context(), llmClient, history, result.Text)
            if suggestionErr != nil {
                log.Printf("Suggestion generation failed: %v", suggestionErr)
            }
        }()
        
        wg.Add(1)
        go func() {
            defer wg.Done()
            llmResponse, responseErr = generateAssistantResponse(r.Context(), llmClient, history, result.Text)
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
        
        // Only show suggestion if it's meaningfully different from the user's text
        if suggestion != "" {
            log.Printf("Raw user text: %s", result.Text)
            log.Printf("Raw suggestion: %s", suggestion)
            
            // Normalize both texts for comparison (case insensitive)
            normalizedSuggestion := normalizeTextForComparison(suggestion)
            normalizedUserText := normalizeTextForComparison(result.Text)
            
            log.Printf("Normalized user text: %s", normalizedUserText)
            log.Printf("Normalized suggestion: %s", normalizedSuggestion)
            
            // If they're the same after normalization, clear the suggestion
            if normalizedSuggestion == normalizedUserText {
                log.Printf("Suggestion is the same as user text after normalization, hiding suggestion")
                suggestion = ""
            } else {
                // Check if the suggestion is just a minor punctuation difference
                // Remove all punctuation and compare
                reg := regexp.MustCompile(`[^\w\s]`)
                cleanSuggestion := reg.ReplaceAllString(normalizedSuggestion, "")
                cleanUserText := reg.ReplaceAllString(normalizedUserText, "")
                
                log.Printf("Clean user text: %s", cleanUserText)
                log.Printf("Clean suggestion: %s", cleanSuggestion)
                
                if cleanSuggestion == cleanUserText {
                    log.Printf("Suggestion is only different in punctuation, hiding suggestion")
                    suggestion = ""
                }
            }
        }
        
        // Convert text to speech
        ttsReq := &tts.TTSRequest{
            Text: llmResponse,
        }
        
        ttsResp, err := ttsService.ConvertTextToSpeech(ttsReq)
        if err != nil {
            log.Printf("TTS conversion failed: %v", err)
            // Even if TTS fails, we can still return the text response
            history = append(history, ConversationTurn{
                Role: "user",
                Content: result.Text,
                Suggestion: suggestion,
                UserName: userName,
            })
            history = append(history, ConversationTurn{
                Role: "assistant",
                Content: llmResponse,
            })
            
            w.Header().Set("Content-Type", "application/json")
            json.NewEncoder(w).Encode(map[string]interface{}{
                "status": "partial_success",
                "transcribed_text": result.Text,
                "llm_response": llmResponse,
                "audio_base64": "",
                "history": history,
                "suggestion": suggestion,
                "user_name": userName,
                "error": "TTS service unavailable, text response only",
            })
            return
        }
        
        if ttsResp.Error != "" {
            log.Printf("TTS error: %s", ttsResp.Error)
            // Handle TTS error but still return text response
            history = append(history, ConversationTurn{
                Role: "user",
                Content: result.Text,
                Suggestion: suggestion,
                UserName: userName,
            })
            history = append(history, ConversationTurn{
                Role: "assistant",
                Content: llmResponse,
            })
            
            w.Header().Set("Content-Type", "application/json")
            json.NewEncoder(w).Encode(map[string]interface{}{
                "status": "partial_success",
                "transcribed_text": result.Text,
                "llm_response": llmResponse,
                "audio_base64": "",
                "history": history,
                "suggestion": suggestion,
                "user_name": userName,
                "error": "TTS error: " + ttsResp.Error,
            })
            return
        }
        
        // Use the audio data from TTS response
        audioBase64 := base64.StdEncoding.EncodeToString(ttsResp.AudioData)
        
        // Update history with new turns
        history = append(history, ConversationTurn{
            Role: "user",
            Content: result.Text,
            Suggestion: suggestion,
            UserName: userName,
        })
        history = append(history, ConversationTurn{
            Role: "assistant",
            Content: llmResponse,
        })
        
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]interface{}{
            "status": "success",
            "transcribed_text": result.Text,
            "llm_response": llmResponse,
            "audio_base64": audioBase64,
            "history": history,
            "suggestion": suggestion,
            "user_name": userName,
        })
    }
}

// expandContractions expands common English contractions
func expandContractions(text string) string {
    contractions := map[string]string{
        "i'm":      "i am",
        "you're":   "you are",
        "he's":     "he is",
        "she's":    "she is",
        "it's":     "it is",
        "we're":    "we are",
        "they're":  "they are",
        "that's":   "that is",
        "who's":    "who is",
        "what's":   "what is",
        "where's":  "where is",
        "when's":   "when is",
        "why's":    "why is",
        "how's":    "how is",
        "isn't":    "is not",
        "aren't":   "are not",
        "wasn't":   "was not",
        "weren't":  "were not",
        "haven't":  "have not",
        "hasn't":   "has not",
        "hadn't":   "had not",
        "don't":    "do not",
        "doesn't":  "does not",
        "didn't":   "did not",
        "won't":    "will not",
        "wouldn't": "would not",
        "can't":    "cannot",
        "couldn't": "could not",
        "shouldn't": "should not",
        "mightn't": "might not",
        "mustn't":  "must not",
        "i'd":      "i would",
        "you'd":    "you would",
        "he'd":     "he would",
        "she'd":    "she would",
        "it'd":     "it would",
        "we'd":     "we would",
        "they'd":   "they would",
        "i'll":     "i will",
        "you'll":   "you will",
        "he'll":    "he will",
        "she'll":   "she will",
        "it'll":    "it will",
        "we'll":    "we will",
        "they'll":  "they will",
        "i've":     "i have",
        "you've":   "you have",
        "we've":    "we have",
        "they've":  "they have",
    }
    
    // First normalize all apostrophe types to standard apostrophe
    text = normalizeApostrophes(text)
    
    words := strings.Fields(text)
    for i, word := range words {
        // Remove any punctuation from the word for comparison
        cleanWord := strings.TrimRight(word, ".,!?;")
        if expanded, exists := contractions[strings.ToLower(cleanWord)]; exists {
            words[i] = expanded
        }
    }
    
    return strings.Join(words, " ")
}

// normalizeApostrophes normalizes all types of apostrophes to a standard apostrophe
func normalizeApostrophes(text string) string {
    // Replace all types of apostrophes and quotation marks with standard apostrophe
    apostropheVariants := []string{
        "’", "‘", "`", "´", "ʹ", "ʻ", "ʼ", "ʽ", "ʾ", "ʿ", "ˊ", "ˋ", "˴", "ʹ", "΄", "՚", "׳", "״", "＇", "'",
        "“", "”", "„", "«", "»", "「", "」", "『", "』", "〝", "〞", "〟", "＂",
    }
    
    for _, variant := range apostropheVariants {
        text = strings.ReplaceAll(text, variant, "'")
    }
    
    return text
}

// normalizeTextForComparison normalizes text for comparison (case insensitive)
func normalizeTextForComparison(text string) string {
    // First normalize all apostrophes
    normalized := normalizeApostrophes(text)
    
    // Expand contractions
    normalized = expandContractions(normalized)
    
    // Remove all punctuation and special characters except spaces
    reg := regexp.MustCompile(`[^a-zA-Z0-9\s]`)
    normalized = reg.ReplaceAllString(normalized, "")
    
    // Convert to lowercase
    normalized = strings.ToLower(normalized)
    
    // Trim whitespace and collapse multiple spaces
    normalized = strings.TrimSpace(normalized)
    normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")
    
    return normalized
}
