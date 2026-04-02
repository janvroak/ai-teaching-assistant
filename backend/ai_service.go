package main

import (
	"os"
	"strings"
)

func aiServiceBaseURL() string {
	baseURL := strings.TrimSpace(os.Getenv("AI_SERVICE_BASE_URL"))
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8000"
	}
	return strings.TrimRight(baseURL, "/")
}

