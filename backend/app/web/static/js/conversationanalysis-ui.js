// UI functions for conversation analysis
const ConversationAnalysisUI = {
    // Format feedback with proper HTML structure
    formatFeedback: function(feedbackText) {
        if (!feedbackText) {
            return '<p>No feedback available at this time.</p>';
        }
        
        // Split the feedback into sections based on common patterns
        const sections = feedbackText.split(/\d+\.\s+/).filter(section => section.trim().length > 0);
        
        if (sections.length > 0) {
            let html = '';
            
            sections.forEach((section, index) => {
                const titleMatch = section.match(/^([^:]+):/);
                const title = titleMatch ? titleMatch[1] : `Area ${index + 1}`;
                const content = titleMatch ? section.replace(titleMatch[0], '').trim() : section.trim();
                
                html += `
                    <div class="feedback-point">
                        <h5>${title}</h5>
                        <p>${content}</p>
                    </div>
                `;
            });
            
            return html;
        }
        
        // Fallback: just wrap the entire text in paragraphs
        return `<p>${feedbackText.replace(/\n/g, '</p><p>')}</p>`;
    },
    
    // Display feedback in the UI
    displayFeedback: function(feedback) {
        const feedbackContent = document.getElementById('feedback-content');
        if (feedbackContent) {
            feedbackContent.innerHTML = this.formatFeedback(feedback);
        }
    },
    
    // Show error message
    showError: function(message, elementId) {
        const element = document.getElementById(elementId);
        if (element) {
            element.innerHTML = `<p>${message}</p>`;
        }
    }
};
