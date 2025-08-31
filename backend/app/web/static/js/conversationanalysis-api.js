import { ConversationUtils } from './shared-conversation-utils.js';

// API functions for conversation analysis
const ConversationAnalysisAPI = {
    // Function to fetch feedback from the server
    fetchFeedback: function(conversationHistory) {
        if (!conversationHistory || conversationHistory.length === 0) {
            return Promise.reject(new Error('No conversation available for feedback'));
        }
        
        return fetch('/api/feedback/generate', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ history: conversationHistory }),
            credentials: 'include'
        })
        .then(response => {
            if (!response.ok) {
                throw new Error('Failed to generate feedback');
            }
            return response.json();
        });
    },

    // Function to fetch the latest conversation
    fetchLatestConversation: function() {
        return fetch('/api/conversation/latest', {
            credentials: 'include'
        })
        .then(response => {
            if (!response.ok) {
                throw new Error('No conversation found');
            }
            return response.json();
        });
    }
};

export { ConversationAnalysisAPI };
