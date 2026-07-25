package api

import (
	"bytes"
	"encoding/json"
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

const systemPrompt = `Sos MatematicaRAG, un tutor inteligente de matematicas de la Universidad Nacional de Tucuman (FACE). 
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

			// Get history for context
			historyRows, _ := db.Query(r.Context(),
				`SELECT role, content FROM chat_messages WHERE session_id = $1 ORDER BY created_at ASC`, req.SessionID)
			var history []OpenAIMessage
			if historyRows != nil {
				defer historyRows.Close()
				for historyRows.Next() {
					var m OpenAIMessage
					if historyRows.Scan(&m.Role, &m.Content) == nil {
						history = append(history, m)
					}
				}
			}

			// Build messages with history
			messages := []OpenAIMessage{{Role: "system", Content: systemPrompt}}
			messages = append(messages, history...)

			// Call OpenAI
			apiKey := getAPIKey(db)
			if apiKey == "" {
				http.Error(w, `{"error":"API key no configurada. Agrega OPENAI_API_KEY en Configuracion > API Keys"}`, http.StatusServiceUnavailable)
				return
			}

			model := req.Model
			if model == "" {
				model = "gpt-3.5-turbo"
			}

			reqBody := OpenAIRequest{
				Model:     model,
				Messages:  messages,
				MaxTokens: 1024,
			}
			body, _ := json.Marshal(reqBody)
			httpReq, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
			httpReq.Header.Set("Content-Type", "application/json")
			httpReq.Header.Set("Authorization", "Bearer "+apiKey)

			resp, err := http.DefaultClient.Do(httpReq)
			if err != nil {
				http.Error(w, `{"error":"error al conectar con OpenAI: `+err.Error()+`"}`, http.StatusBadGateway)
				return
			}
			defer resp.Body.Close()

			var openAIResp OpenAIResponse
			json.NewDecoder(resp.Body).Decode(&openAIResp)

			if openAIResp.Error != nil {
				http.Error(w, `{"error":"OpenAI: `+openAIResp.Error.Message+`"}`, http.StatusBadGateway)
				return
			}
			if len(openAIResp.Choices) == 0 {
				http.Error(w, `{"error":"sin respuesta de OpenAI"}`, http.StatusBadGateway)
				return
			}

			response := strings.TrimSpace(openAIResp.Choices[0].Message.Content)

			var assistantID string
			err = db.QueryRow(r.Context(),
				`INSERT INTO chat_messages (session_id, role, content, model) VALUES ($1, 'ASSISTANT', $2, $3) RETURNING id`,
				req.SessionID, response, model,
			).Scan(&assistantID)
			if err != nil {
				http.Error(w, `{"error":"failed to save response"}`, http.StatusInternalServerError)
				return
			}
			_, _ = db.Exec(r.Context(), `UPDATE chat_sessions SET updated_at = NOW() WHERE id = $1`, req.SessionID)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ChatMessage{
				ID: assistantID, Role: "ASSISTANT", Content: response,
				Model: model, CreatedAt: time.Now(),
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
