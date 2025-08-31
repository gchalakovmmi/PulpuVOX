// Conversation recording and processing logic
let mediaRecorder;
let audioChunks = [];
let stream;
let conversationHistory = [];
let isFirstTurn = true;
let isRecording = false;
let userName = "You"; // Default value
let recordingTimeout;
let shouldEndAfterProcessing = false;

// Get references to UI elements
const startButton = document.getElementById('startButton');
const endConversationButton = document.getElementById('endConversationButton');
const statusIndicator = document.getElementById('statusIndicator');
const conversationHistoryDiv = document.getElementById('conversation-history');
const audioPlayer = document.getElementById('audio-player');

// Hello sound URL
const helloSoundUrl = "/static/audio/hello.mp3";

// Function to update UI state
function updateUIState(state) {
    switch(state) {
        case 'ready':
            startButton.disabled = false;
            startButton.textContent = 'Start Conversation';
            startButton.classList.remove('btn-danger', 'btn-success');
            startButton.classList.add('btn-primary');
            statusIndicator.textContent = 'Ready to start';
            endConversationButton.disabled = true;
            break;
        case 'recording':
            startButton.disabled = false;
            startButton.textContent = 'End Turn';
            startButton.classList.remove('btn-primary');
            startButton.classList.add('btn-primary'); // Blue for end turn
            statusIndicator.innerHTML = '<span class="indicator pulse"></span> Recording... Speak now';
            endConversationButton.disabled = false;
            break;
        case 'processing':
            startButton.disabled = true;
            startButton.textContent = 'Processing...';
            statusIndicator.innerHTML = '<span class="indicator"></span> Processing...';
            break;
        case 'playing':
            startButton.disabled = true;
            startButton.textContent = 'End Turn';
            statusIndicator.innerHTML = '<span class="indicator"></span> Playing response...';
            break;
        case 'preparing':
            startButton.disabled = true;
            startButton.textContent = 'Preparing...';
            statusIndicator.textContent = 'Preparing for next turn...';
            break;
    }
}

// Function to update the message display
function updateMessageDisplay() {
    conversationHistoryDiv.innerHTML = '';
    
    conversationHistory.forEach(turn => {
        const messageDiv = document.createElement('div');
        messageDiv.className = turn.role === 'user' ? 'message user-message' : 'message assistant-message';
        
        const roleSpan = document.createElement('span');
        roleSpan.className = 'message-role';
        
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
        
        messageDiv.appendChild(roleSpan);
        messageDiv.appendChild(contentSpan);
        
        // Add suggestion if available
        if (turn.suggestion !== undefined && turn.suggestion !== '') {
            // Show suggestion for incorrect sentences
            const suggestionDiv = document.createElement('div');
            suggestionDiv.className = 'suggestion';
            suggestionDiv.innerHTML = '<strong>Suggestion:</strong> ' + turn.suggestion;
            messageDiv.appendChild(suggestionDiv);
        } else if (turn.role === 'user' && turn.suggestion !== undefined) {
            // Show positive feedback for correct sentences (empty suggestion means correct)
            const positiveDiv = document.createElement('div');
            positiveDiv.className = 'positive-feedback';
            positiveDiv.innerHTML = '✓ Good job! Your sentence is correct.';
            messageDiv.appendChild(positiveDiv);
        } else if (turn.role === 'user' && turn.isProcessing) {
            // Show processing indicator for user messages that are still being processed
            const processingDiv = document.createElement('div');
            processingDiv.className = 'processing';
            processingDiv.innerHTML = 'Processing...';
            messageDiv.appendChild(processingDiv);
        }
        
        conversationHistoryDiv.appendChild(messageDiv);
    });
    
    // Scroll to bottom to show latest message
    conversationHistoryDiv.scrollTop = conversationHistoryDiv.scrollHeight;
}

// Function to stop recording
function stopRecording() {
    if (mediaRecorder && mediaRecorder.state !== 'inactive') {
        mediaRecorder.stop();
        updateUIState('processing');
        // Clear the timeout if recording is stopped manually
        if (recordingTimeout) {
            clearTimeout(recordingTimeout);
            recordingTimeout = null;
        }
    }
}

// Function to start recording process
async function startRecordingProcess() {
    try {
        if (isRecording) return; // Prevent multiple recordings
        isRecording = true;
        
        // Update UI to show we're preparing to record
        startButton.disabled = true;
        startButton.textContent = 'Preparing...';
        statusIndicator.innerHTML = '<span class="indicator"></span> Preparing...';
        
        // Stop any existing stream
        if (stream) {
            stream.getTracks().forEach(track => track.stop());
        }
        
        // Request access to the microphone
        stream = await navigator.mediaDevices.getUserMedia({
            audio: {
                channelCount: 1,
                sampleRate: 44100,
                sampleSize: 16
            }
        });
        
        // Create a media recorder instance
        mediaRecorder = new MediaRecorder(stream);
        
        // Reset audio chunks
        audioChunks = [];
        
        // Event handler for when data is available
        mediaRecorder.ondataavailable = (event) => {
            if (event.data.size > 0) {
                audioChunks.push(event.data);
            }
        };
        
        // Event handler for when recording stops
        mediaRecorder.onstop = () => {
            statusIndicator.innerHTML = '<span class="indicator"></span> Processing...';
            // Create a blob from the audio chunks
            const audioBlob = new Blob(audioChunks, { type: 'audio/wav' });
            // Convert to MP3
            convertToMp3(audioBlob);
            isRecording = false;
        };
        
        // Start recording
        mediaRecorder.start();
        updateUIState('recording');
        
        // Set a timeout to automatically stop recording after 30 seconds
        recordingTimeout = setTimeout(() => {
            if (isRecording) {
                statusIndicator.innerHTML = '<span class="indicator"></span> Time limit reached, processing...';
                stopRecording();
            }
        }, 30000); // 30 seconds
        
    } catch (error) {
        console.error("Error:", error);
        statusIndicator.textContent = "Error: " + error.message;
        updateUIState('ready');
        isRecording = false;
        // Clear the timeout if there was an error
        if (recordingTimeout) {
            clearTimeout(recordingTimeout);
            recordingTimeout = null;
        }
    }
}

// Event handler for the start/stop button
startButton.addEventListener('click', function() {
    if (isRecording) {
        stopRecording();
    } else {
        startConversation();
    }
});

function startConversation() {
    try {
        updateUIState('processing');
        
        // If it's the first turn, add the hello message immediately
        if (isFirstTurn) {
            const helloMessage = {
                role: 'assistant',
                content: "Hello! What would you like to talk about today?"
            };
            conversationHistory.push(helloMessage);
            updateMessageDisplay();
            isFirstTurn = false;
        }
        
        // Play the hello sound
        const helloSound = new Audio(helloSoundUrl);
        helloSound.onended = async () => {
            await startRecordingProcess();
        };
        helloSound.onerror = (error) => {
            console.error("Error playing hello sound:", error);
            statusIndicator.textContent = "Error playing hello sound";
            updateUIState('ready');
        };
        helloSound.play().catch(error => {
            console.error("Failed to play hello sound:", error);
            statusIndicator.textContent = "Failed to play hello sound";
            updateUIState('ready');
        });
        
    } catch (error) {
        console.error("Error:", error);
        statusIndicator.textContent = "Error: " + error.message;
        updateUIState('ready');
    }
}

// Event handler for the end conversation button
endConversationButton.addEventListener('click', function(e) {
    // Prevent any default behavior
    e.preventDefault();
    e.stopPropagation();
    
    // Always allow ending the conversation, even during recording
    endConversation();
});

function endConversation() {
    // If we're currently recording, stop the recording first
    if (isRecording) {
        stopRecording();
        
        // Set a flag to indicate we want to end after processing
        shouldEndAfterProcessing = true;
        return;
    }
    
    // Save conversation to sessionStorage for the analysis page
    sessionStorage.setItem('currentConversation', JSON.stringify(conversationHistory));
    
    fetch('/api/conversation/end', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ history: conversationHistory }),
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
        statusIndicator.textContent = "Error ending conversation: " + error.message;
        updateUIState('ready');
    });
}

// Function to convert audio to MP3 using lamejs
function convertToMp3(blob) {
    // Create a file reader to read the blob
    const reader = new FileReader();
    reader.onload = function() {
        // Create audio context to decode audio
        const audioContext = new (window.AudioContext || window.webkitAudioContext)();
        audioContext.decodeAudioData(reader.result, function(buffer) {
            // Get the PCM data from the buffer
            const pcmData = buffer.getChannelData(0);
            const sampleRate = buffer.sampleRate;
            // Initialize the MP3 encoder
            const mp3Encoder = new lamejs.Mp3Encoder(1, sampleRate, 128);
            // Convert float32 to int16
            const samples = new Int16Array(pcmData.length);
            for (let i = 0; i < pcmData.length; i++) {
                samples[i] = pcmData[i] * 32767;
            }
            // Encode the PCM data to MP3
            const mp3Data = [];
            const sampleBlockSize = 1152;
            for (let i = 0; i < samples.length; i += sampleBlockSize) {
                const chunk = samples.subarray(i, i + sampleBlockSize);
                const mp3Buffer = mp3Encoder.encodeBuffer(chunk);
                if (mp3Buffer.length > 0) {
                    mp3Data.push(mp3Buffer);
                }
            }
            // Finalize the encoding
            const finalBuffer = mp3Encoder.flush();
            if (finalBuffer.length > 0) {
                mp3Data.push(finalBuffer);
            }
            // Combine all MP3 data
            const combined = new Uint8Array(mp3Data.reduce((acc, val) => {
                const newArray = new Uint8Array(acc.length + val.length);
                newArray.set(acc);
                newArray.set(val, acc.length);
                return newArray;
            }, new Uint8Array()));
            // Create a blob from the MP3 data
            const mp3Blob = new Blob([combined], { type: 'audio/mp3' });
            // Send to server for transcription and TTS
            sendToServer(mp3Blob);
        }, function(error) {
            console.error("Error decoding audio:", error);
            statusIndicator.textContent = "Error processing audio";
            updateUIState('ready');
        });
    };
    reader.onerror = function(error) {
        console.error("Error reading blob:", error);
        statusIndicator.textContent = "Error processing recording";
        updateUIState('ready');
    };
    // Read the blob as array buffer
    reader.readAsArrayBuffer(blob);
}

// Function to send MP3 to server
function sendToServer(mp3Blob) {
    const formData = new FormData();
    formData.append('audio', mp3Blob, 'recording.mp3');
    formData.append('history', JSON.stringify(conversationHistory));

    // Add a temporary user message with "Processing..." indicator
    conversationHistory.push({
        role: 'user',
        content: 'Processing...',
        isProcessing: true
    });
    updateMessageDisplay();

    fetch('/api/conversation/turn', {
        method: 'POST',
        body: formData,
        credentials: 'include'  // This ensures cookies are sent with the request
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
        conversationHistory.pop();
        
        // Update the conversation history with the real data
        conversationHistory = data.history;
        
        // Update the user name if provided
        if (data.user_name) {
            userName = data.user_name;
        }
        
        // Update the message display
        updateMessageDisplay();
        
        // Save conversation to sessionStorage for the analysis page
        sessionStorage.setItem('currentConversation', JSON.stringify(conversationHistory));
        
        // Check if we need to end the conversation after processing
        if (shouldEndAfterProcessing) {
            shouldEndAfterProcessing = false;
            // Call endConversation again to actually end it
            endConversation();
            return;
        }
        
        // Check if we have audio
        if (data.audio_base64 && data.status !== "partial_success") {
            const audio = new Audio("data:audio/mp3;base64," + data.audio_base64);
            audio.onended = function() {
                updateUIState('preparing');
                // Reset for next recording
                audioChunks = [];
                // Automatically start the next recording after a short delay
                setTimeout(() => {
                    startRecordingProcess();
                }, 1000); // 1 second delay before starting next recording
            };
            audio.onerror = function(e) {
                console.error("Audio playback failed:", e);
                updateUIState('preparing');
                // Reset for next recording
                audioChunks = [];
                setTimeout(() => {
                    startRecordingProcess();
                }, 1000);
            };
            audio.play().catch(e => {
                console.error("Audio play error:", e);
                updateUIState('preparing');
                // Reset for next recording
                audioChunks = [];
                setTimeout(() => {
                    startRecordingProcess();
                }, 1000);
            });
            updateUIState('playing');
        } else {
            // No audio available, but we can still continue
            updateUIState('preparing');
            // Reset for next recording
            audioChunks = [];
            setTimeout(() => {
                startRecordingProcess();
            }, 1000);
        }
    })
    .catch(error => {
        console.error('Error sending to server:', error);
        // Remove the temporary processing message
        conversationHistory.pop();
        updateMessageDisplay();
        statusIndicator.textContent = "Error: " + error.message;
        updateUIState('ready');
    });
}

// Initialize UI state
updateUIState('ready');
updateMessageDisplay();
