package main

import (
    "log"

    "PulpuVOX/internal/config"
    "PulpuVOX/internal/logger"
    "PulpuVOX/internal/server"
)

func main() {
    // Load configuration
    cfg := config.Load()
    
    // Initialize JSON logging
    log.SetFlags(0)
    log.SetOutput(&logger.JSONLogger{Instance: cfg.InstanceName})
    
    log.Printf("Serving on port %s...\n", cfg.Port)

    // Create and start server
    srv := server.New(cfg)
    log.Fatal(srv.ListenAndServe())
}
