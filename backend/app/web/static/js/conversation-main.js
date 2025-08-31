// Main application logic for conversation
document.addEventListener('DOMContentLoaded', function() {
    // Initialize UI state
    ConversationUI.updateUIState('ready');
    ConversationUI.updateMessageDisplay();
    
    // Event handler for the start/stop button
    ConversationUI.elements.startButton.addEventListener('click', function() {
        if (ConversationState.getIsRecording()) {
            ConversationRecording.stopRecording();
        } else {
            startConversation();
        }
    });
    
    // Event handler for the end conversation button
    ConversationUI.elements.endConversationButton.addEventListener('click', function(e) {
        // Prevent any default behavior
        e.preventDefault();
        e.stopPropagation();
        
        // Always allow ending the conversation, even during recording
        ConversationAPI.endConversation();
    });
    
    // Function to start conversation
    async function startConversation() {
        try {
            ConversationUI.updateUIState('processing');
            
            // If it's the first turn, add the hello message immediately
            if (ConversationState.getIsFirstTurn()) {
                ConversationState.addToConversationHistory({
                    role: 'assistant',
                    content: "Hello! What would you like to talk about today?"
                });
                ConversationUI.updateMessageDisplay();
                ConversationState.setIsFirstTurn(false);
            }
            
            // Play the hello sound
            await ConversationUI.playHelloSound();
            await ConversationRecording.startRecordingProcess();
            
        } catch (error) {
            console.error("Error starting conversation:", error);
            ConversationUI.elements.statusIndicator.textContent = "Error: " + error.message;
            ConversationUI.updateUIState('ready');
        }
    }
});
