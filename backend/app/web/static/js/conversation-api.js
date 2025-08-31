import { ConversationState } from './conversation-state.js';
import { ConversationUI } from './conversation-ui.js';
import { ConversationRecording } from './conversation-recording.js';
import { CONSTANTS } from './constants.js';

// API communication for conversation
export const ConversationAPI = {
    // Function to send MP3 to server
    sendToServer: function(mp3Blob) {
        // Check if we should skip processing (for immediate end conversation)
        if (ConversationState.getShouldSkipProcessing()) {
            ConversationState.setShouldSkipProcessing(false);
            return; // Don't process, just return
        }
        
        const formData = new FormData();
        formData.append('audio', mp3Blob, 'recording.mp3');
        formData.append('history', JSON.stringify(ConversationState.getConversationHistory()));
        
        // Add a temporary user message with "Processing..." indicator
        ConversationState.addToConversationHistory({
            role: 'user',
            content: 'Processing...',
            isProcessing: true
        });
        
        ConversationUI.updateMessageDisplay();
        
        fetch('/api/conversation/turn', {
            method: 'POST',
            body: formData,
            credentials: 'include'
        })
        .then(response => {
            if (!response.ok) {
                throw new Error('Server returned an error: ' + response.status);
            }
            return response.json();
        })
        .then(data => {
            console.log('API response:', data);
            
            // Remove the temporary processing message
            ConversationState.removeLastFromConversationHistory();
            
            // Update the conversation history with the real data
            ConversationState.setConversationHistory(data.history);
            
            // Update the user name if provided
            if (data.user_name) {
                ConversationState.setUserName(data.user_name);
            }
            
            // Update the message display
            ConversationUI.updateMessageDisplay();
            
            // Save conversation to sessionStorage for the analysis page
            sessionStorage.setItem('currentConversation', JSON.stringify(ConversationState.getConversationHistory()));
            
            // Check if we need to end the conversation after processing
            if (ConversationState.getShouldEndAfterProcessing()) {
                ConversationState.setShouldEndAfterProcessing(false);
                // Call endConversation again to actually end it
                this.endConversation();
                return;
            }
            
            // Check if we have audio
            if (data.audio_base64 && data.status !== "partial_success") {
                const audio = new Audio("data:audio/mp3;base64," + data.audio_base64);
                
                audio.onended = () => {
                    ConversationUI.updateUIState(CONSTANTS.UI_STATES.PREPARING);
                    // Reset for next recording
                    ConversationState.resetAudioChunks();
                    
                    // Automatically start the next recording after a short delay
                    setTimeout(() => {
                        ConversationRecording.startRecordingProcess();
                    }, 1000);
                };
                
                audio.onerror = (e) => {
                    console.error("Audio playback failed:", e);
                    ConversationUI.updateUIState(CONSTANTS.UI_STATES.PREPARING);
                    // Reset for next recording
                    ConversationState.resetAudioChunks();
                    
                    setTimeout(() => {
                        ConversationRecording.startRecordingProcess();
                    }, 1000);
                };
                
                audio.play().catch(e => {
                    console.error("Audio play error:", e);
                    ConversationUI.updateUIState(CONSTANTS.UI_STATES.PREPARING);
                    // Reset for next recording
                    ConversationState.resetAudioChunks();
                    
                    setTimeout(() => {
                        ConversationRecording.startRecordingProcess();
                    }, 1000);
                });
                
                ConversationUI.updateUIState(CONSTANTS.UI_STATES.PLAYING);
            } else {
                // No audio available, but we can still continue
                ConversationUI.updateUIState(CONSTANTS.UI_STATES.PREPARING);
                // Reset for next recording
                ConversationState.resetAudioChunks();
                
                setTimeout(() => {
                    ConversationRecording.startRecordingProcess();
                }, 1000);
            }
        })
        .catch(error => {
            console.error('Error sending to server:', error);
            // Remove the temporary processing message
            ConversationState.removeLastFromConversationHistory();
            ConversationUI.updateMessageDisplay();
            ConversationUI.elements.statusIndicator.textContent = "Error: " + error.message;
            ConversationUI.updateUIState(CONSTANTS.UI_STATES.READY);
        });
    },

    // Function to end conversation
    endConversation: function() {
        // If we're currently recording, stop the recording and skip processing
        if (ConversationState.getIsRecording()) {
            ConversationState.setShouldSkipProcessing(true);
            ConversationRecording.stopRecordingAndCleanup();
            
            // Small delay to allow the recording to stop completely
            setTimeout(() => {
                this.sendEndRequest();
            }, 100);
            return;
        }
        
        // If not recording, send the end request immediately
        this.sendEndRequest();
    },
    
    // Helper function to send the end conversation request
    sendEndRequest: function() {
        // Update UI state to show we're processing the end conversation request
        ConversationUI.updateUIState(CONSTANTS.UI_STATES.PROCESSING);
        
        // Save conversation to sessionStorage for the analysis page
        sessionStorage.setItem('currentConversation', JSON.stringify(ConversationState.getConversationHistory()));
        
        fetch('/api/conversation/end', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ history: ConversationState.getConversationHistory() }),
            credentials: 'include'
        })
        .then(response => {
            if (!response.ok) {
                throw new Error('Failed to end conversation');
            }
            return response.json();
        })
        .then(data => {
            window.location.href = data.redirect;
        })
        .catch(error => {
            console.error('Error ending conversation:', error);
            ConversationUI.elements.statusIndicator.textContent = "Error ending conversation: " + error.message;
            ConversationUI.updateUIState(CONSTANTS.UI_STATES.READY);
            ConversationUI.elements.endConversationButton.disabled = false;
            ConversationUI.elements.endConversationButton.innerHTML = '<i class="fas fa-stop-circle me-2"></i>End Conversation';
        });
    }
};
