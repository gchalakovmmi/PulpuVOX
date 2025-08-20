package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"mime/multipart"
	"net/http"
)

// TranscribeService handles audio transcription
type TranscribeService struct {
	WhisperURL string
}

// NewTranscribeService creates a new transcription service
func NewTranscribeService(whisperURL string) *TranscribeService {
	return &TranscribeService{
		WhisperURL: whisperURL,
	}
}

// TranscribeRequest represents a transcription request
type TranscribeRequest struct {
	AudioData     []byte
	FileName      string
	Language      string
	Task          string
	OutputFormat  string
	ShouldEncode  bool
}

// TranscribeResponse represents a transcription response
type TranscribeResponse struct {
	Text      string      `json:"text"`
	Segments  []Segment   `json:"segments"`
	Language  string      `json:"language"`
	Error     string      `json:"error,omitempty"`
}

// Segment represents a segment of the transcribed text
type Segment struct {
	ID               int     `json:"id"`
	Seek             int     `json:"seek"`
	Start            float64 `json:"start"`
	End              float64 `json:"end"`
	Text             string  `json:"text"`
	Tokens           []int   `json:"tokens"`
	Temperature      float64 `json:"temperature"`
	AvgLogprob       float64 `json:"avg_logprob"`
	CompressionRatio float64 `json:"compression_ratio"`
	NoSpeechProb     float64 `json:"no_speech_prob"`
}

// ParseAudioFromRequest extracts audio data from an HTTP request
func ParseAudioFromRequest(r *http.Request) ([]byte, string, error) {
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB max
		return nil, "", err
	}

	file, header, err := r.FormFile("audio")
	if err != nil {
		return nil, "", err
	}
	defer file.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, file); err != nil {
		return nil, "", err
	}

	return buf.Bytes(), header.Filename, nil
}

// SendToWhisper forwards audio data to the Whisper service
func (ts *TranscribeService) SendToWhisper(req *TranscribeRequest) (*TranscribeResponse, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("audio_file", req.FileName)
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(part, bytes.NewReader(req.AudioData)); err != nil {
		return nil, err
	}
	writer.Close()

	// Build the URL with query parameters
	whisperURL := ts.WhisperURL + "?encode=true&task=transcribe&language=en&output=json"
	
	httpReq, err := http.NewRequest("POST", whisperURL, body)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result TranscribeResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// TranscribeHandler handles HTTP requests for transcription
func (ts *TranscribeService) TranscribeHandler(w http.ResponseWriter, r *http.Request) {
	audioData, fileName, err := ParseAudioFromRequest(r)
	if err != nil {
		http.Error(w, "Unable to process audio: "+err.Error(), http.StatusBadRequest)
		return
	}

	req := &TranscribeRequest{
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

	// Log the transcribed text
	log.Printf("Transcribed text: %s", result.Text)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
