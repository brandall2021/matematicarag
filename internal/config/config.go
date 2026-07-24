package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Port            string
	DatabaseURL     string
	JWTSecret       string
	OpenAIAPIKey    string
	QdrantHost      string
	QdrantPort      string
	CORSOrigins     string
	EmbeddingType   string
	Environment     string
}

func Load() *Config {
	return &Config{
		Port:          getEnv("PORT", "8009"),
		DatabaseURL:   getEnv("DB_URL", "postgresql://localhost:5432/matematica"),
		JWTSecret:     getEnvRequired("JWT_SECRET"),
		OpenAIAPIKey:  getEnvRequired("OPENAI_API_KEY"),
		QdrantHost:    getEnv("QDRANT_HOST", "localhost"),
		QdrantPort:    getEnv("QDRANT_PORT", "6334"),
		CORSOrigins:   getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:4200"),
		EmbeddingType: getEnv("EMBEDDING_TYPE", "openai"),
		Environment:   getEnv("APP_ENV", "development"),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvRequired(key string) string {
	val := os.Getenv(key)
	if val == "" {
		panic(fmt.Sprintf("Required environment variable %s is not set", key))
	}
	return val
}

func (c *Config) CORSOriginsList() []string {
	return strings.Split(c.CORSOrigins, ",")
}
