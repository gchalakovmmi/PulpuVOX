import { ConversationState } from './conversation-state.js';
import { ConversationUI } from './conversation-ui.js';
import { ConversationRecording } from './conversation-recording.js';
import { CONSTANTS } from './constants.js';

// API communication for conversation
const ConversationAPI = {
    // Function to send MP3 to server
    sendToServer: function(mp3Blob) {
        // Check if we should skip processing (for immediate end conversation)
        if (ConversationState.getShouldSkipProcessing()) {
            ConversationState.setShouldSkipProcessing(false);
            this.endConversation();
            return;
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
        // If we're currently recording, stop the recording first
        if (ConversationState.getIsRecording()) {
            ConversationState.setShouldSkipProcessing(true);
            ConversationRecording.stopRecording();
            return;
        }
        
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
        });
    }
};

// Export the ConversationAPI object
export { ConversationAPI };
