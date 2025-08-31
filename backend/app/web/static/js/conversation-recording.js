// Recording functionality for conversation
const ConversationRecording = {
    // Function to stop recording
    stopRecording: function() {
        const mediaRecorder = ConversationState.getMediaRecorder();
        if (mediaRecorder && mediaRecorder.state !== 'inactive') {
            mediaRecorder.stop();
            ConversationUI.updateUIState('processing');
            ConversationState.clearRecordingTimeout();
        }
    },
    
    // Function to start recording process
    startRecordingProcess: async function() {
        try {
            if (ConversationState.getIsRecording()) return; // Prevent multiple recordings
            
            ConversationState.setIsRecording(true);
            ConversationUI.updateUIState('recording');
            
            // Stop any existing stream
            ConversationState.setStream(null);
            
            // Request access to the microphone
            const stream = await navigator.mediaDevices.getUserMedia({
                audio: {
                    channelCount: 1,
                    sampleRate: 44100,
                    sampleSize: 16
                }
            });
            
            ConversationState.setStream(stream);
            
            // Create a media recorder instance
            const mediaRecorder = new MediaRecorder(stream);
            ConversationState.setMediaRecorder(mediaRecorder);
            
            // Reset audio chunks
            ConversationState.resetAudioChunks();
            
            // Event handler for when data is available
            mediaRecorder.ondataavailable = (event) => {
                if (event.data.size > 0) {
                    const audioChunks = ConversationState.getAudioChunks();
                    audioChunks.push(event.data);
                    ConversationState.setAudioChunks(audioChunks);
                }
            };
            
            // Event handler for when recording stops
            mediaRecorder.onstop = () => {
                ConversationUI.updateUIState('processing');
                // Create a blob from the audio chunks
                const audioBlob = new Blob(ConversationState.getAudioChunks(), { type: 'audio/wav' });
                // Convert to MP3
                this.convertToMp3(audioBlob);
                ConversationState.setIsRecording(false);
            };
            
            // Start recording
            mediaRecorder.start();
            
            // Set a timeout to automatically stop recording after 30 seconds
            const recordingTimeout = setTimeout(() => {
                if (ConversationState.getIsRecording()) {
                    ConversationUI.elements.statusIndicator.innerHTML = '<span class="indicator"></span> Time limit reached, processing...';
                    this.stopRecording();
                }
            }, 30000); // 30 seconds
            
            ConversationState.setRecordingTimeout(recordingTimeout);
            
        } catch (error) {
            console.error("Error starting recording:", error);
            ConversationUI.elements.statusIndicator.textContent = "Error: " + error.message;
            ConversationUI.updateUIState('ready');
            ConversationState.setIsRecording(false);
            ConversationState.clearRecordingTimeout();
        }
    },
    
    // Function to convert audio to MP3 using lamejs
    convertToMp3: function(blob) {
        // Check if we should skip processing (for immediate end conversation)
        if (ConversationState.getShouldSkipProcessing()) {
            ConversationState.setShouldSkipProcessing(false);
            ConversationAPI.endConversation();
            return;
        }
        
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
                ConversationAPI.sendToServer(mp3Blob);
            }, function(error) {
                console.error("Error decoding audio:", error);
                ConversationUI.elements.statusIndicator.textContent = "Error processing audio";
                ConversationUI.updateUIState('ready');
            });
        };
        reader.onerror = function(error) {
            console.error("Error reading blob:", error);
            ConversationUI.elements.statusIndicator.textContent = "Error processing recording";
            ConversationUI.updateUIState('ready');
        };
        // Read the blob as array buffer
        reader.readAsArrayBuffer(blob);
    }
};
