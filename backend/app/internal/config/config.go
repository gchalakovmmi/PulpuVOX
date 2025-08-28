package config

import (
    "os"
    "time"
)

type Config struct {
    Port         string
    ReadTimeout  time.Duration
    WriteTimeout time.Duration
    IdleTimeout  time.Duration
    InstanceName string
}

func Load() Config {
    return Config{
        Port:         getEnv("BACKEND_PORT", "8080"),
        ReadTimeout:  getEnvAsDuration("READ_TIMEOUT", 15*time.Second),
        WriteTimeout: getEnvAsDuration("WRITE_TIMEOUT", 15*time.Second),
        IdleTimeout:  getEnvAsDuration("IDLE_TIMEOUT", 60*time.Second),
        InstanceName: getEnv("INSTANCE_NAME", "myapp-1"),
    }
}

func getEnv(key, defaultValue string) string {
    if value, exists := os.LookupEnv(key); exists {
        return value
    }
    return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
    if value, exists := os.LookupEnv(key); exists {
        if dur, err := time.ParseDuration(value); err == nil {
            return dur
        }
    }
    return defaultValue
}
