package logger

import (
    "encoding/json"
    "fmt"
    "time"
)

type JSONLogger struct {
    Instance string
}

func (l *JSONLogger) Write(p []byte) (n int, err error) {
    logEntry := map[string]interface{}{
        "timestamp": time.Now().UTC().Format(time.RFC3339),
        "level":     "info",
        "instance":  l.Instance,
        "message":   string(p),
    }
    
    jsonBytes, err := json.Marshal(logEntry)
    if err != nil {
        return 0, err
    }
    
    // Write to stdout with a newline
    fmt.Println(string(jsonBytes))
    return len(p), nil
}
