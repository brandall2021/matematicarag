package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SendMessageRequest struct {
	SessionID string `json:"sessionId"`
	Content   string `json:"content"`
	Model     string `json:"model"`
}

type ChatMessage struct {
	ID        string          `json:"id"`
	Role      string          `json:"role"`
	Content   string          `json:"content"`
	Model     string          `json:"model"`
	Sources   json.RawMessage `json:"sources,omitempty"`
	CreatedAt time.Time       `json:"createdAt"`
}

type ChatSession struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

const defaultChatPrompt = `Sos MatematicaRAG, un tutor inteligente de matematicas de la Universidad Nacional de Tucuman (FACE). 
Respondes en español, de forma clara y didactica. Explicas conceptos matematicos con ejemplos.
Si te hacen una pregunta de matematicas, resuelvela paso a paso.
Si te saludan, responde de forma amigable.
Sos experto en: algebra, calculo, geometria, estadistica, probabilidad, lineal algebra, ecuaciones diferenciales.`

func ChatRoutes(db *pgxpool.Pool, cfg *config.Config) func(r chi.Router) {
	return func(r chi.Router) {
		r.Use(AuthMiddleware(cfg.JWTSecret))
		r.Post("/message", func(w http.ResponseWriter, r *http.Request) {
			userID := r.Context().Value(UserIDKey).(string)
			var req SendMessageRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
				return
			}
			if req.Content == "" {
				http.Error(w, `{"error":"content is required"}`, http.StatusBadRequest)
				return
			}
			if req.SessionID == "" {
				var sessionID string
				err := db.QueryRow(r.Context(),
					`INSERT INTO chat_sessions (user_id, title) VALUES ($1, $2) RETURNING id`,
					userID, truncate(req.Content, 100),
				).Scan(&sessionID)
				if err != nil {
					http.Error(w, `{"error":"failed to create session"}`, http.StatusInternalServerError)
					return
				}
				req.SessionID = sessionID
			}

			_, err := db.Exec(r.Context(),
				`INSERT INTO chat_messages (session_id, role, content, model) VALUES ($1, 'USER', $2, $3)`,
				req.SessionID, req.Content, req.Model,
			)
			if err != nil {
				http.Error(w, `{"error":"failed to save message"}`, http.StatusInternalServerError)
				return
			}

			historyRows, _ := db.Query(r.Context(),
				`SELECT role, content FROM chat_messages WHERE session_id = $1 ORDER BY created_at ASC`, req.SessionID)
			var history []OpenAIMessage
			if historyRows != nil {
				defer historyRows.Close()
				for historyRows.Next() {
					var m OpenAIMessage
					if historyRows.Scan(&m.Role, &m.Content) == nil {
						m.Role = strings.ToLower(m.Role)
						history = append(history, m)
					}
				}
			}

			ragSources, ragContext := performRAGSearch(db, req.Content)

			customPrompt := getSetting(db, "CHAT_SYSTEM_PROMPT")
			if customPrompt == "" {
				customPrompt = defaultChatPrompt
			}

			if ragContext != "" {
				customPrompt += "\n\nTenes acceso a documentos de referencia. Usa la siguiente informacion para responder cuando sea relevante. Cita las fuentes usando el formato: [Fuente: nombre_archivo, pagina X]."
			}

			var messages []OpenAIMessage
			if ragContext != "" {
				messages = []OpenAIMessage{
					{Role: "system", Content: customPrompt},
				}
				messages = append(messages, history...)
				messages = append(messages, OpenAIMessage{Role: "user", Content: req.Content + "\n\nContexto de documentos:\n" + ragContext})
			} else {
				messages = []OpenAIMessage{{Role: "system", Content: customPrompt}}
				messages = append(messages, history...)
				messages = append(messages, OpenAIMessage{Role: "user", Content: req.Content})
			}

			model := getModel(db, req.Model)
			apiKey := getAPIKey(db)
			if apiKey == "" {
				http.Error(w, `{"error":"API key no configurada. Agrega tu key en Configuracion > API Keys"}`, http.StatusServiceUnavailable)
				return
			}

			response, callErr := callOpenAIWithHistory(db, messages, model)

			if callErr != nil {
				http.Error(w, `{"error":"`+callErr.Error()+`"}`, http.StatusBadGateway)
				return
			}

			var sourcesJSON json.RawMessage
			if len(ragSources) > 0 {
				sourcesJSON, _ = json.Marshal(ragSources)
			}

			var assistantID string
			err = db.QueryRow(r.Context(),
				`INSERT INTO chat_messages (session_id, role, content, model, sources) VALUES ($1, 'ASSISTANT', $2, $3, $4) RETURNING id`,
				req.SessionID, response, model, sourcesJSON,
			).Scan(&assistantID)
			if err != nil {
				http.Error(w, `{"error":"failed to save response"}`, http.StatusInternalServerError)
				return
			}
			_, _ = db.Exec(r.Context(), `UPDATE chat_sessions SET updated_at = NOW() WHERE id = $1`, req.SessionID)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ChatMessage{
				ID: assistantID, Role: "ASSISTANT", Content: response,
				Model: model, Sources: sourcesJSON, CreatedAt: time.Now(),
			})
		})

		r.Get("/sessions", func(w http.ResponseWriter, r *http.Request) {
			userID := r.Context().Value(UserIDKey).(string)
			rows, err := db.Query(r.Context(),
				`SELECT id, title, created_at, updated_at FROM chat_sessions WHERE user_id = $1 ORDER BY updated_at DESC LIMIT 50`, userID,
			)
			if err != nil {
				http.Error(w, `{"error":"failed to list sessions"}`, http.StatusInternalServerError)
				return
			}
			defer rows.Close()
			sessions := make([]ChatSession, 0)
			for rows.Next() {
				var s ChatSession
				if err := rows.Scan(&s.ID, &s.Title, &s.CreatedAt, &s.UpdatedAt); err == nil {
					sessions = append(sessions, s)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(sessions)
		})

		r.Get("/sessions/{id}/messages", func(w http.ResponseWriter, r *http.Request) {
			userID := r.Context().Value(UserIDKey).(string)
			sessionID := chi.URLParam(r, "id")
			var exists bool
			db.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM chat_sessions WHERE id = $1 AND user_id = $2)`, sessionID, userID).Scan(&exists)
			if !exists {
				http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
				return
			}
			rows, err := db.Query(r.Context(),
				`SELECT id, role, content, COALESCE(model, ''), COALESCE(sources, '[]'), created_at FROM chat_messages WHERE session_id = $1 ORDER BY created_at ASC`, sessionID,
			)
			if err != nil {
				http.Error(w, `{"error":"failed to get messages"}`, http.StatusInternalServerError)
				return
			}
			defer rows.Close()
			messages := make([]ChatMessage, 0)
			for rows.Next() {
				var m ChatMessage
				if err := rows.Scan(&m.ID, &m.Role, &m.Content, &m.Model, &m.Sources, &m.CreatedAt); err == nil {
					messages = append(messages, m)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(messages)
		})

		r.Delete("/sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
			userID := r.Context().Value(UserIDKey).(string)
			sessionID := chi.URLParam(r, "id")
			result, err := db.Exec(r.Context(), `DELETE FROM chat_sessions WHERE id = $1 AND user_id = $2`, sessionID, userID)
			if err != nil {
				http.Error(w, `{"error":"failed to delete session"}`, http.StatusInternalServerError)
				return
			}
			if result.RowsAffected() == 0 {
				http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

func performRAGSearch(db *pgxpool.Pool, query string) ([]RagSource, string) {
	results, err := VectorSearch(db, query, 5)
	if err != nil || len(results) == 0 {
		return nil, ""
	}

	var contextParts []string
	sources := make([]RagSource, len(results))
	for i, res := range results {
		sourceLabel := buildSourceLabel(res.Filename, res.Page, res.Section)
		contextParts = append(contextParts, fmt.Sprintf("[Fuente: %s]\n%s", sourceLabel, res.Content))
		sources[i] = RagSource{
			ID:       res.ChunkID,
			Content:  truncateString(res.Content, 300),
			Score:    res.Score,
			Filename: res.Filename,
			Page:     res.Page,
			Section:  res.Section,
			URL:      res.URL,
		}
	}

	return sources, strings.Join(contextParts, "\n---\n")
}
