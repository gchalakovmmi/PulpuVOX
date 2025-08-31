import { ConversationUtils } from './shared-conversation-utils.js';
import { ConversationAnalysisAPI } from './conversationanalysis-api.js';
import { ConversationAnalysisUI } from './conversationanalysis-ui.js';

// Main application logic for conversation analysis
document.addEventListener('DOMContentLoaded', function() {
    // Try to get conversation from sessionStorage first
    const conversationHistory = ConversationUtils.loadConversationFromStorage();
    
    if (conversationHistory && conversationHistory.length > 0) {
        ConversationUtils.displayConversation(conversationHistory, 'conversation-history');
        
        // Fetch and display feedback
        ConversationAnalysisAPI.fetchFeedback(conversationHistory)
            .then(data => {
                ConversationAnalysisUI.displayFeedback(data.feedback);
            })
            .catch(error => {
                console.error('Error fetching feedback:', error);
                ConversationAnalysisUI.showError('Unable to generate feedback at this time. Please try again later.', 'feedback-content');
            });
        
        return;
    }
    
    // Fall back to API call if no conversation in sessionStorage
    ConversationAnalysisAPI.fetchLatestConversation()
        .then(history => {
            if (history && history.length > 0) {
                ConversationUtils.displayConversation(history, 'conversation-history');
                
                // Fetch and display feedback
                ConversationAnalysisAPI.fetchFeedback(history)
                    .then(data => {
                        ConversationAnalysisUI.displayFeedback(data.feedback);
                    })
                    .catch(error => {
                        console.error('Error fetching feedback:', error);
                        ConversationAnalysisUI.showError('Unable to generate feedback at this time. Please try again later.', 'feedback-content');
                    });
            } else {
                ConversationUtils.displayConversation([], 'conversation-history');
                ConversationAnalysisUI.showError('No conversation available for feedback.', 'feedback-content');
            }
        })
        .catch(error => {
            console.error('Error fetching conversation:', error);
            ConversationUtils.displayConversation([], 'conversation-history');
            ConversationAnalysisUI.showError('Unable to load conversation for feedback.', 'feedback-content');
        });
});
