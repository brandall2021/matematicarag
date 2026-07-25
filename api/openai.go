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
	Model     string          `json:"model"`
	Messages  []OpenAIMessage `json:"messages"`
	MaxTokens int            `json:"max_tokens,omitempty"`
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

type AnthropicRequest struct {
	Model     string              `json:"model"`
	MaxTokens int                 `json:"max_tokens"`
	System    string              `json:"system,omitempty"`
	Messages  []AnthropicMessage  `json:"messages"`
}

type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AnthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func getAPIKey(db *pgxpool.Pool) string {
	keyName := getSetting(db, "AI_API_KEY_NAME")
	if keyName == "" {
		keyName = "OPENAI_API_KEY"
	}
	return getSetting(db, keyName)
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

func getProvider(db *pgxpool.Pool) string {
	p := getSetting(db, "AI_PROVIDER")
	if p == "" {
		p = "openai"
	}
	return p
}

func getModel(db *pgxpool.Pool, override string) string {
	if override != "" {
		return override
	}
	m := getSetting(db, "AI_MODEL")
	if m != "" {
		return m
	}
	return "gpt-3.5-turbo"
}

func callOpenAI(db *pgxpool.Pool, systemPrompt string, userMessage string, model string) (string, error) {
	return callOpenAIWithHistory(db, []OpenAIMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMessage},
	}, model)
}

func callOpenAIWithHistory(db *pgxpool.Pool, messages []OpenAIMessage, model string) (string, error) {
	provider := getProvider(db)
	model = getModel(db, model)
	apiKey := getAPIKey(db)

	if apiKey == "" {
		return "", fmt.Errorf("no API key configured. Add your key in Configuracion > API Keys")
	}

	if provider == "anthropic" {
		return callAnthropicWithMessages(apiKey, model, messages)
	}

	return callOpenAICompatibleWithMessages(provider, apiKey, model, messages)
}

func callOpenAICompatible(provider, apiKey, model, systemPrompt, userMessage string) (string, error) {
	return callOpenAICompatibleWithMessages(provider, apiKey, model, []OpenAIMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMessage},
	})
}

func callOpenAICompatibleWithMessages(provider, apiKey, model string, messages []OpenAIMessage) (string, error) {
	var baseURL string
	switch provider {
	case "groq":
		baseURL = "https://api.groq.com/openai/v1/chat/completions"
	case "openrouter":
		baseURL = "https://openrouter.ai/api/v1/chat/completions"
	default:
		baseURL = "https://api.openai.com/v1/chat/completions"
	}

	reqBody := OpenAIRequest{
		Model:     model,
		Messages:  messages,
		MaxTokens: 1024,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", baseURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error calling %s: %v", provider, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var openAIResp OpenAIResponse
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return "", fmt.Errorf("invalid response from %s", provider)
	}

	if openAIResp.Error != nil {
		return "", fmt.Errorf("%s error: %s", provider, openAIResp.Error.Message)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("no response from %s", provider)
	}

	return strings.TrimSpace(openAIResp.Choices[0].Message.Content), nil
}

func callAnthropic(apiKey, model, systemPrompt, userMessage string) (string, error) {
	return callAnthropicWithMessages(apiKey, model, []OpenAIMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMessage},
	})
}

func callAnthropicWithMessages(apiKey, model string, messages []OpenAIMessage) (string, error) {
	var systemMsg string
	var anthropicMessages []AnthropicMessage
	for _, m := range messages {
		if m.Role == "system" {
			systemMsg = m.Content
		} else {
			anthropicMessages = append(anthropicMessages, AnthropicMessage{Role: m.Role, Content: m.Content})
		}
	}
	if len(anthropicMessages) == 0 {
		anthropicMessages = []AnthropicMessage{{Role: "user", Content: "hello"}}
	}

	reqBody := AnthropicRequest{
		Model:     model,
		MaxTokens: 1024,
		System:    systemMsg,
		Messages:  anthropicMessages,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error calling Anthropic: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var anthropicResp AnthropicResponse
	if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
		return "", fmt.Errorf("invalid response from Anthropic")
	}

	if anthropicResp.Error != nil {
		return "", fmt.Errorf("Anthropic error: %s", anthropicResp.Error.Message)
	}

	if len(anthropicResp.Content) == 0 {
		return "", fmt.Errorf("no response from Anthropic")
	}

	return strings.TrimSpace(anthropicResp.Content[0].Text), nil
}
