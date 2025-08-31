import { ConversationState } from './conversation-state.js';
import { CONSTANTS } from './constants.js';

// UI management for conversation
const ConversationUI = {
    // DOM elements
    elements: {
        startButton: null,
        endConversationButton: null,
        statusIndicator: null,
        conversationHistoryDiv: null,
        audioPlayer: null
    },

    // Initialize UI elements
    init: function() {
        this.elements.startButton = document.getElementById('startButton');
        this.elements.endConversationButton = document.getElementById('endConversationButton');
        this.elements.statusIndicator = document.getElementById('statusIndicator');
        this.elements.conversationHistoryDiv = document.getElementById('conversation-history');
        this.elements.audioPlayer = document.getElementById('audio-player');
        
        // Show initial placeholder if no conversation history
        if (ConversationState.getConversationHistory().length === 0) {
            this.showPlaceholder();
        }
    },

    // Show placeholder content
    showPlaceholder: function() {
        if (this.elements.conversationHistoryDiv) {
            this.elements.conversationHistoryDiv.innerHTML = `
                <div class="text-center text-muted py-5">
                    <i class="fas fa-comments fa-3x mb-2"></i>
                    <p>Your conversation will appear here</p>
                </div>
            `;
        }
    },

    // Function to update UI state
    updateUIState: function(state) {
        if (!this.elements.startButton || !this.elements.statusIndicator) return;
        
        switch(state) {
            case CONSTANTS.UI_STATES.READY:
                this.elements.startButton.disabled = false;
                this.elements.startButton.innerHTML = '<i class="fas fa-microphone me-2"></i>Start Conversation';
                this.elements.startButton.classList.remove('btn-danger', 'btn-success');
                this.elements.startButton.classList.add('btn-primary');
                this.elements.statusIndicator.textContent = 'Ready to start';
                this.elements.endConversationButton.disabled = true;
                break;
            case CONSTANTS.UI_STATES.RECORDING:
                this.elements.startButton.disabled = false;
                this.elements.startButton.innerHTML = '<i class="fas fa-stop me-2"></i>End Turn';
                this.elements.startButton.classList.remove('btn-primary');
                this.elements.startButton.classList.add('btn-danger');
                this.elements.statusIndicator.innerHTML = '<span class="indicator pulse"></span> Recording... Speak now';
                this.elements.endConversationButton.disabled = false;
                break;
            case CONSTANTS.UI_STATES.PROCESSING:
                this.elements.startButton.disabled = true;
                this.elements.startButton.innerHTML = '<i class="fas fa-spinner fa-spin me-2"></i>Processing...';
                this.elements.statusIndicator.innerHTML = '<span class="indicator"></span> Processing...';
                break;
            case CONSTANTS.UI_STATES.PLAYING:
                this.elements.startButton.disabled = true;
                this.elements.startButton.innerHTML = '<i class="fas fa-volume-up me-2"></i>Playing...';
                this.elements.statusIndicator.innerHTML = '<span class="indicator"></span> Playing response...';
                break;
            case CONSTANTS.UI_STATES.PREPARING:
                this.elements.startButton.disabled = true;
                this.elements.startButton.innerHTML = '<i class="fas fa-cog fa-spin me-2"></i>Preparing...';
                this.elements.statusIndicator.textContent = 'Preparing for next turn...';
                break;
        }
    },

    // Function to update the message display
    updateMessageDisplay: function() {
        const conversationHistory = ConversationState.getConversationHistory();
        
        // If no history, show placeholder
        if (conversationHistory.length === 0) {
            this.showPlaceholder();
            return;
        }
        
        // Clear and rebuild conversation display
        this.elements.conversationHistoryDiv.innerHTML = '';
        
        conversationHistory.forEach(turn => {
            const messageDiv = document.createElement('div');
            messageDiv.className = turn.role === 'user' ? 'message user-message' : 'message assistant-message';
            
            const roleSpan = document.createElement('span');
            roleSpan.className = 'message-role';
            
            if (turn.role === 'user') {
                roleSpan.textContent = (turn.user_name || ConversationState.getUserName()) + ': ';
            } else if (turn.role === 'assistant') {
                roleSpan.textContent = 'Voxy: ';
            } else {
                roleSpan.textContent = turn.role + ': ';
            }
            
            const contentSpan = document.createElement('span');
            contentSpan.className = 'message-content';
            contentSpan.textContent = turn.content;
            
            messageDiv.appendChild(roleSpan);
            messageDiv.appendChild(contentSpan);
            
            // Add suggestion if available
            if (turn.suggestion !== undefined && turn.suggestion !== '') {
                const suggestionDiv = document.createElement('div');
                suggestionDiv.className = 'suggestion';
                suggestionDiv.innerHTML = '<strong>Suggestion:</strong> ' + turn.suggestion;
                messageDiv.appendChild(suggestionDiv);
            } else if (turn.role === 'user' && turn.suggestion !== undefined) {
                const positiveDiv = document.createElement('div');
                positiveDiv.className = 'positive-feedback';
                positiveDiv.innerHTML = '✓ Good job! Your sentence is correct.';
                messageDiv.appendChild(positiveDiv);
            } else if (turn.role === 'user' && turn.isProcessing) {
                const processingDiv = document.createElement('div');
                processingDiv.className = 'processing';
                processingDiv.innerHTML = 'Processing...';
                messageDiv.appendChild(processingDiv);
            }
            
            this.elements.conversationHistoryDiv.appendChild(messageDiv);
        });
        
        // Scroll to bottom to show latest message
        this.elements.conversationHistoryDiv.scrollTop = this.elements.conversationHistoryDiv.scrollHeight;
    },

    // Play hello sound
    playHelloSound: function() {
        return new Promise((resolve, reject) => {
            const helloSound = new Audio(CONSTANTS.HELLO_SOUND_URL);
            helloSound.onended = resolve;
            helloSound.onerror = reject;
            helloSound.play().catch(reject);
        });
    }
};

// Export the ConversationUI object
export { ConversationUI };
