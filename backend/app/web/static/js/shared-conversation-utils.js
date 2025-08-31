// Shared utility functions for conversation features
const ConversationUtils = {
    // Format message with role and content
    formatMessage: function(turn, userName = "You") {
        const div = document.createElement('div');
        div.className = turn.role === 'user' ? 'user-message' : 'assistant-message';
        
        const roleSpan = document.createElement('strong');
        
        if (turn.role === 'user') {
            roleSpan.textContent = (turn.user_name || userName) + ': ';
        } else if (turn.role === 'assistant') {
            roleSpan.textContent = 'Voxy: ';
        } else {
            roleSpan.textContent = turn.role + ': ';
        }
        
        const contentSpan = document.createElement('span');
        contentSpan.className = 'message-content';
        contentSpan.textContent = turn.content;
        
        div.appendChild(roleSpan);
        div.appendChild(contentSpan);
        
        // Add suggestion if available
        if (turn.suggestion !== undefined && turn.suggestion !== null) {
            if (turn.suggestion === '') {
                // Show positive feedback for correct sentences
                const positiveDiv = document.createElement('div');
                positiveDiv.className = 'positive-feedback';
                positiveDiv.innerHTML = '✓ Good job! Your sentence is correct.';
                div.appendChild(positiveDiv);
            } else {
                // Show suggestion for incorrect sentences
                const suggestionDiv = document.createElement('div');
                suggestionDiv.className = 'suggestion';
                suggestionDiv.innerHTML = '<strong>Suggestion:</strong> ' + turn.suggestion;
                div.appendChild(suggestionDiv);
            }
        } else if (turn.role === 'user' && turn.isProcessing) {
            // Show processing indicator for user messages that are still being processed
            const processingDiv = document.createElement('div');
            processingDiv.className = 'processing';
            processingDiv.innerHTML = 'Processing...';
            div.appendChild(processingDiv);
        }
        
        return div;
    },
    
    // Display conversation history
    displayConversation: function(history, containerId, userName = "You") {
        const container = document.getElementById(containerId);
        if (!container) {
            console.error(`Container with ID ${containerId} not found`);
            return;
        }
        
        container.innerHTML = ''; // Clear loading message
        
        if (!history || history.length === 0) {
            container.innerHTML = '<div class="text-center text-muted py-3">No conversation history found.</div>';
            return;
        }
        
        history.forEach(turn => {
            const messageElement = this.formatMessage(turn, userName);
            container.appendChild(messageElement);
        });
        
        // Scroll to bottom to show latest message
        container.scrollTop = container.scrollHeight;
    },
    
    // Save conversation to sessionStorage
    saveConversationToStorage: function(conversationHistory) {
        try {
            sessionStorage.setItem('currentConversation', JSON.stringify(conversationHistory));
            return true;
        } catch (error) {
            console.error('Error saving conversation to storage:', error);
            return false;
        }
    },
    
    // Load conversation from sessionStorage
    loadConversationFromStorage: function() {
        try {
            const conversationHistory = JSON.parse(sessionStorage.getItem('currentConversation'));
            return conversationHistory || [];
        } catch (error) {
            console.error('Error loading conversation from storage:', error);
            return [];
        }
    }
};
