package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIRequest struct {
	Model    string          `json:"model"`
	Messages []OpenAIMessage `json:"messages"`
	MaxTokens int           `json:"max_tokens,omitempty"`
}

type OpenAIChoice struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}

type OpenAIResponse struct {
	Choices []OpenAIChoice `json:"choices"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func getAPIKey(db *pgxpool.Pool) string {
	var key string
	err := db.QueryRow(context.Background(),
		`SELECT value FROM app_settings WHERE key = 'OPENAI_API_KEY'`).Scan(&key)
	if err == nil && key != "" {
		return key
	}
	return ""
}

func getSetting(db *pgxpool.Pool, key string) string {
	var val string
	err := db.QueryRow(context.Background(),
		`SELECT value FROM app_settings WHERE key = $1`, key).Scan(&val)
	if err == nil {
		return val
	}
	return ""
}

func callOpenAI(db *pgxpool.Pool, systemPrompt string, userMessage string, model string) (string, error) {
	apiKey := getAPIKey(db)
	if apiKey == "" {
		return "", fmt.Errorf("no API key configured. Add OPENAI_API_KEY in Configuracion > API Keys")
	}
	if model == "" {
		model = "gpt-3.5-turbo"
	}

	reqBody := OpenAIRequest{
		Model: model,
		Messages: []OpenAIMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMessage},
		},
		MaxTokens: 1024,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error calling OpenAI: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var openAIResp OpenAIResponse
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return "", err
	}

	if openAIResp.Error != nil {
		return "", fmt.Errorf("OpenAI error: %s", openAIResp.Error.Message)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return strings.TrimSpace(openAIResp.Choices[0].Message.Content), nil
}
