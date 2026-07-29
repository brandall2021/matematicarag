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
	// RAG Configuration
	ChunkSize       int
	ChunkOverlap    int
	VectorWeight    float64
	KeywordWeight   float64
	RetrievalTopK   int
	RerankTopK      int
	RAGMinScore     float64
	EnableHybrid    bool
	EnableReranker  bool
	EnableCitations bool
	// Math Service
	MathServiceURL string
	MathTimeout    int
	// Adaptive Engine
	AdaptiveHintWeight      float64
	AdaptiveErrorWeight     float64
	AdaptiveMasteryThreshold float64
	AdaptiveMaxDifficulty   int
	// Assessment Engine
	AssessmentDefaultTimeLimit  int
	AssessmentMaxAttempts       int
	AssessmentPassingScore      float64
	AssessmentAutoGradeEnabled  bool
	AssessmentRecoveryThreshold float64
	AssessmentAlertThreshold    float64
	// Agent settings
	AgentMaxToolCalls       int
	AgentMaxRetries         int
	AgentIntentThreshold    float64
	AgentLowMastery         float64
	AgentHighMastery        float64
	AgentManualReviewThresh float64
}

func Load() *Config {
	return &Config{
		Port:          getEnv("PORT", "8008"),
		DatabaseURL:   getEnv("DB_URL", "postgresql://localhost:5432/matematica"),
		JWTSecret:     getEnvRequired("JWT_SECRET"),
		OpenAIAPIKey:  getEnvRequired("OPENAI_API_KEY"),
		QdrantHost:    getEnv("QDRANT_HOST", "localhost"),
		QdrantPort:    getEnv("QDRANT_PORT", "6334"),
		CORSOrigins:   getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:4200"),
		EmbeddingType: getEnv("EMBEDDING_TYPE", "openai"),
		Environment:   getEnv("APP_ENV", "development"),
		// RAG defaults
		ChunkSize:       getEnvInt("CHUNK_SIZE", 500),
		ChunkOverlap:    getEnvInt("CHUNK_OVERLAP", 50),
		VectorWeight:    getEnvFloat("VECTOR_WEIGHT", 0.60),
		KeywordWeight:   getEnvFloat("KEYWORD_WEIGHT", 0.40),
		RetrievalTopK:   getEnvInt("RETRIEVAL_TOP_K", 20),
		RerankTopK:      getEnvInt("RERANK_TOP_K", 5),
		RAGMinScore:     getEnvFloat("RAG_MIN_SCORE", 0.70),
		EnableHybrid:    getEnvBool("ENABLE_HYBRID_SEARCH", true),
		EnableReranker:  getEnvBool("ENABLE_RERANKER", true),
		EnableCitations: getEnvBool("ENABLE_CITATIONS", true),
		// Math Service
		MathServiceURL: getEnv("MATH_SERVICE_URL", "http://localhost:5000"),
		MathTimeout:    getEnvInt("MATH_TIMEOUT", 5),
		// Adaptive Engine
		AdaptiveHintWeight:      getEnvFloat("ADAPTIVE_HINT_WEIGHT", 0.1),
		AdaptiveErrorWeight:     getEnvFloat("ADAPTIVE_ERROR_WEIGHT", 0.03),
		AdaptiveMasteryThreshold: getEnvFloat("ADAPTIVE_MASTERY_THRESHOLD", 0.8),
		AdaptiveMaxDifficulty:   getEnvInt("ADAPTIVE_MAX_DIFFICULTY", 5),
		// Assessment Engine
		AssessmentDefaultTimeLimit:  getEnvInt("ASSESSMENT_DEFAULT_TIME_LIMIT", 60),
		AssessmentMaxAttempts:       getEnvInt("ASSESSMENT_MAX_ATTEMPTS", 3),
		AssessmentPassingScore:      getEnvFloat("ASSESSMENT_PASSING_SCORE", 0.6),
		AssessmentAutoGradeEnabled:  getEnvBool("ASSESSMENT_AUTO_GRADE_ENABLED", true),
		AssessmentRecoveryThreshold: getEnvFloat("ASSESSMENT_RECOVERY_THRESHOLD", 0.6),
		AssessmentAlertThreshold:    getEnvFloat("ASSESSMENT_ALERT_THRESHOLD", 0.4),
		// Agent settings
		AgentMaxToolCalls:       getEnvInt("AGENT_MAX_TOOL_CALLS", 8),
		AgentMaxRetries:         getEnvInt("AGENT_MAX_RETRIES", 2),
		AgentIntentThreshold:    getEnvFloat("AGENT_INTENT_THRESHOLD", 0.75),
		AgentLowMastery:         getEnvFloat("AGENT_LOW_MASTERY", 0.40),
		AgentHighMastery:        getEnvFloat("AGENT_HIGH_MASTERY", 0.70),
		AgentManualReviewThresh: getEnvFloat("AGENT_MANUAL_REVIEW_THRESHOLD", 0.65),
	}
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			return n
		}
	}
	return defaultVal
}

func getEnvFloat(key string, defaultVal float64) float64 {
	if v := os.Getenv(key); v != "" {
		var f float64
		if _, err := fmt.Sscanf(v, "%f", &f); err == nil {
			return f
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if v := os.Getenv(key); v != "" {
		return v == "true" || v == "1" || v == "yes"
	}
	return defaultVal
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
