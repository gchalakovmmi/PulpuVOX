import { CONSTANTS } from './constants.js';

// State management for conversation
export const ConversationState = {
    mediaRecorder: null,
    audioChunks: [],
    stream: null,
    conversationHistory: [],
    isFirstTurn: true,
    isRecording: false,
    userName: "You",
    recordingTimeout: null,
    shouldEndAfterProcessing: false,
    shouldSkipProcessing: false,

    // Getters and setters for controlled state access
    getMediaRecorder: function() { return this.mediaRecorder; },
    setMediaRecorder: function(recorder) { this.mediaRecorder = recorder; },
    getAudioChunks: function() { return this.audioChunks; },
    setAudioChunks: function(chunks) { this.audioChunks = chunks; },
    resetAudioChunks: function() { this.audioChunks = []; },
    getStream: function() { return this.stream; },
    setStream: function(newStream) {
        // Clean up previous stream if exists
        if (this.stream) {
            this.stream.getTracks().forEach(track => track.stop());
        }
        this.stream = newStream;
    },
    getConversationHistory: function() { return this.conversationHistory; },
    setConversationHistory: function(history) { this.conversationHistory = history; },
    addToConversationHistory: function(turn) { this.conversationHistory.push(turn); },
    removeLastFromConversationHistory: function() { return this.conversationHistory.pop(); },
    getIsFirstTurn: function() { return this.isFirstTurn; },
    setIsFirstTurn: function(value) { this.isFirstTurn = value; },
    getIsRecording: function() { return this.isRecording; },
    setIsRecording: function(value) { this.isRecording = value; },
    getUserName: function() { return this.userName; },
    setUserName: function(name) { this.userName = name; },
    getRecordingTimeout: function() { return this.recordingTimeout; },
    setRecordingTimeout: function(timeout) { this.recordingTimeout = timeout; },
    clearRecordingTimeout: function() {
        if (this.recordingTimeout) {
            clearTimeout(this.recordingTimeout);
            this.recordingTimeout = null;
        }
    },
    getShouldEndAfterProcessing: function() { return this.shouldEndAfterProcessing; },
    setShouldEndAfterProcessing: function(value) { this.shouldEndAfterProcessing = value; },
    getShouldSkipProcessing: function() { return this.shouldSkipProcessing; },
    setShouldSkipProcessing: function(value) { this.shouldSkipProcessing = value; },

    // Reset all state (useful for ending conversation)
    reset: function() {
        this.mediaRecorder = null;
        this.audioChunks = [];
        if (this.stream) {
            this.stream.getTracks().forEach(track => track.stop());
        }
        this.stream = null;
        this.isRecording = false;
        this.recordingTimeout = null;
        this.shouldEndAfterProcessing = false;
        this.shouldSkipProcessing = false;
    }
};
