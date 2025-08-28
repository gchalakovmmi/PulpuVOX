package services

import (
    "log"

    "PulpuVOX/internal/openai"
    "PulpuVOX/internal/tts"
    "PulpuVOX/internal/whisper"
)

type Services struct {
    WhisperService *whisper.TranscribeService
    OpenAIClient   *openai.Client
    TTSService     *tts.TTSService
}

func New() *Services {
    // Initialize Whisper service
    whisperService, err := whisper.NewTranscribeService()
    if err != nil {
        log.Fatalf("Failed to initialize Whisper service: %v", err)
    }

    // Initialize OpenAI client
    openaiClient, err := openai.NewClient()
    if err != nil {
        log.Fatalf("Failed to initialize OpenAI client: %v", err)
    }

    // Initialize TTS service
    ttsService, err := tts.NewTTSService()
    if err != nil {
        log.Fatalf("Failed to initialize TTS service: %v", err)
    }

    return &Services{
        WhisperService: whisperService,
        OpenAIClient:   openaiClient,
        TTSService:     ttsService,
    }
}
