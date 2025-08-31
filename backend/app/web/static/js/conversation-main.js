import { ConversationState } from './conversation-state.js';
import { ConversationUI } from './conversation-ui.js';
import { ConversationRecording } from './conversation-recording.js';
import { ConversationAPI } from './conversation-api.js';
import { CONSTANTS } from './constants.js';

// Main application logic for conversation
document.addEventListener('DOMContentLoaded', function() {
    // Initialize UI elements
    ConversationUI.init();
    
    // Initialize UI state
    ConversationUI.updateUIState(CONSTANTS.UI_STATES.READY);
    
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
        
        // Update UI immediately to provide feedback
        ConversationUI.elements.endConversationButton.disabled = true;
        ConversationUI.elements.endConversationButton.innerHTML = '<i class="fas fa-spinner fa-spin me-2"></i>Ending...';
        
        // Always allow ending the conversation, even during recording
        ConversationAPI.endConversation();
    });

    // Function to start conversation
    async function startConversation() {
        try {
            ConversationUI.updateUIState(CONSTANTS.UI_STATES.PROCESSING);
            
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
            ConversationUI.updateUIState(CONSTANTS.UI_STATES.READY);
        }
    }
});
