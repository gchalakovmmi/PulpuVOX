package feedback

import (
    "encoding/json"
    "log"
    "net/http"
    "strings"

    "PulpuVOX/internal/handlers/conversation"
    "PulpuVOX/internal/openai"
    "github.com/jackc/pgx/v5"
)

func GenerateFeedbackHandler(w http.ResponseWriter, r *http.Request, conn *pgx.Conn) {
    var request struct {
        History []conversation.ConversationTurn `json:"history"`
    }

    if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Initialize OpenAI client
    llmClient, err := openai.NewClient()
    if err != nil {
        log.Printf("Failed to create OpenAI client: %v", err)
        http.Error(w, "Failed to initialize feedback service", http.StatusInternalServerError)
        return
    }

    // Build the conversation context for the prompt
    var conversationContext strings.Builder
    for _, turn := range request.History {
        if turn.Role == "user" {
            conversationContext.WriteString("Student: " + turn.Content + "\n")
            if turn.Suggestion != "" {
                conversationContext.WriteString("Corrected: " + turn.Suggestion + "\n")
            }
        } else if turn.Role == "assistant" {
            conversationContext.WriteString("Teacher: " + turn.Content + "\n")
        }
    }

    // Create the prompt for feedback generation
    prompt := `You are an experienced English teacher. Below is a conversation between a student and a teacher. 
The student's turns include their original text and a corrected version when applicable.

Please analyze the conversation and provide constructive feedback on the student's English proficiency.
Focus on:
1. Recurring grammatical errors
2. Pronunciation issues (based on the transcriptions)
3. Vocabulary usage and suggestions for improvement
4. Sentence structure and fluency
5. Overall communication effectiveness
6. Grade the level of the student example A1, A2, B1, B2, C1 or C2

Provide specific examples from the conversation and suggestions for what the student should focus on to improve.

Conversation:
` + conversationContext.String() + `

Please provide your feedback:`

    messages := []openai.ChatCompletionMessage{
        {
            Role:    "system",
            Content: "You are a helpful and encouraging English teacher providing constructive feedback to students.",
        },
        {
            Role:    "user",
            Content: prompt,
        },
    }

    // Send to LLM for feedback generation
    chatCompletion, err := llmClient.CreateChatCompletion(&openai.ChatCompletionRequest{
        Messages: messages,
        Model:    llmClient.Model,
    })
    if err != nil {
        log.Printf("Feedback generation failed: %v", err)
        http.Error(w, "Feedback generation failed", http.StatusInternalServerError)
        return
    }

    feedback := chatCompletion.Choices[0].Message.Content
    log.Printf("Generated feedback: %s", feedback)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "feedback": feedback,
    })
}
