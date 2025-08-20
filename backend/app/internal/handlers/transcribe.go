package handlers

import (
	"github.com/gchalakovmmi/PulpuWEB/whisper"
	"encoding/json"
	"log"
	"net/http"
)

func TranscribeHandler(ts *whisper.TranscribeService) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        audioData, fileName, err := whisper.ParseAudioFromRequest(r)
        if err != nil {
            http.Error(w, "Unable to process audio: "+err.Error(), http.StatusBadRequest)
            return
        }

        req := &whisper.TranscribeRequest{
            AudioData:    audioData,
            FileName:     fileName,
            Language:     "en",
            Task:         "transcribe",
            OutputFormat: "json",
            ShouldEncode: true,
        }

        result, err := ts.SendToWhisper(req)
        if err != nil {
            http.Error(w, "Transcription failed: "+err.Error(), http.StatusInternalServerError)
            return
        }

        log.Printf("Transcribed text: %s", result.Text)
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]string{"status": "success"})
    }
}
