// UI management for conversation
const ConversationUI = {
    // DOM elements
    elements: {
        startButton: document.getElementById('startButton'),
        endConversationButton: document.getElementById('endConversationButton'),
        statusIndicator: document.getElementById('statusIndicator'),
        conversationHistoryDiv: document.getElementById('conversation-history'),
        audioPlayer: document.getElementById('audio-player')
    },
    
    // Hello sound URL
    helloSoundUrl: "/static/audio/hello.mp3",
    
    // Function to update UI state
    updateUIState: function(state) {
        switch(state) {
            case 'ready':
                this.elements.startButton.disabled = false;
                this.elements.startButton.textContent = 'Start Conversation';
                this.elements.startButton.classList.remove('btn-danger', 'btn-success');
                this.elements.startButton.classList.add('btn-primary');
                this.elements.statusIndicator.textContent = 'Ready to start';
                this.elements.endConversationButton.disabled = true;
                break;
            case 'recording':
                this.elements.startButton.disabled = false;
                this.elements.startButton.textContent = 'End Turn';
                this.elements.startButton.classList.remove('btn-primary');
                this.elements.startButton.classList.add('btn-primary');
                this.elements.statusIndicator.innerHTML = '<span class="indicator pulse"></span> Recording... Speak now';
                this.elements.endConversationButton.disabled = false;
                break;
            case 'processing':
                this.elements.startButton.disabled = true;
                this.elements.startButton.textContent = 'Processing...';
                this.elements.statusIndicator.innerHTML = '<span class="indicator"></span> Processing...';
                break;
            case 'playing':
                this.elements.startButton.disabled = true;
                this.elements.startButton.textContent = 'End Turn';
                this.elements.statusIndicator.innerHTML = '<span class="indicator"></span> Playing response...';
                break;
            case 'preparing':
                this.elements.startButton.disabled = true;
                this.elements.startButton.textContent = 'Preparing...';
                this.elements.statusIndicator.textContent = 'Preparing for next turn...';
                break;
        }
    },
    
    // Function to update the message display
    updateMessageDisplay: function() {
        const conversationHistory = ConversationState.getConversationHistory();
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
            const helloSound = new Audio(this.helloSoundUrl);
            helloSound.onended = resolve;
            helloSound.onerror = reject;
            helloSound.play().catch(reject);
        });
    }
};
