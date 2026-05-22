package credential

import (
	"errors"
	"fmt"
	"strings"
)

const ollamaDefaultBaseURL = "http://localhost:11434/v1"

// OllamaDefaultBaseURL is the default OpenAI-compatible Ollama endpoint.
const OllamaDefaultBaseURL = ollamaDefaultBaseURL

// NormalizeOllamaOpenAIBaseURL ensures the URL ends with /v1.
func NormalizeOllamaOpenAIBaseURL(baseURL string) string {
	if baseURL == "" {
		return ""
	}
	trimmed := strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(trimmed, "/v1") {
		return trimmed
	}
	return trimmed + "/v1"
}

var errOllamaNoModels = errors.New("ollama is running but no models are installed — run: ollama pull llama3.2")

// FormatOllamaConnectError turns probe/network failures into actionable setup hints.
func FormatOllamaConnectError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, errOllamaNoModels) {
		return err
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "connection refused"),
		strings.Contains(msg, "connect: connection refused"),
		strings.Contains(msg, "no such host"),
		strings.Contains(msg, "network is unreachable"):
		return fmt.Errorf("cannot reach Ollama — make sure it is running (ollama serve) and the URL is correct")
	case strings.Contains(msg, "context deadline exceeded"),
		strings.Contains(msg, "timeout"),
		strings.Contains(msg, "i/o timeout"):
		return fmt.Errorf("ollama timed out — check the URL and that ollama serve is running")
	default:
		return fmt.Errorf("ollama connection failed: %w", err)
	}
}
