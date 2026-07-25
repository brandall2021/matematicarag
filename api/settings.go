package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Setting struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

func SettingsRoutes(db *pgxpool.Pool) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			rows, err := db.Query(r.Context(), `SELECT key, value, COALESCE(description, '') FROM app_settings ORDER BY key`)
			if err != nil {
				http.Error(w, `{"error":"failed to list settings"}`, http.StatusInternalServerError)
				return
			}
			defer rows.Close()
			settings := make([]Setting, 0)
			for rows.Next() {
				var s Setting
				if err := rows.Scan(&s.Key, &s.Value, &s.Description); err == nil {
					settings = append(settings, s)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(settings)
		})

		r.Get("/{key}", func(w http.ResponseWriter, r *http.Request) {
			key := chi.URLParam(r, "key")
			var s Setting
			err := db.QueryRow(r.Context(), `SELECT key, value, COALESCE(description, '') FROM app_settings WHERE key = $1`, key).Scan(&s.Key, &s.Value, &s.Description)
			if err != nil {
				http.Error(w, `{"error":"setting not found"}`, http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(s)
		})

		r.Put("/{key}", func(w http.ResponseWriter, r *http.Request) {
			key := chi.URLParam(r, "key")
			var req Setting
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
				return
			}
			_, err := db.Exec(r.Context(),
				`INSERT INTO app_settings (key, value, description, updated_at) VALUES ($1, $2, $3, NOW())
				 ON CONFLICT (key) DO UPDATE SET value = $2, description = $3, updated_at = NOW()`,
				key, req.Value, req.Description,
			)
			if err != nil {
				http.Error(w, `{"error":"failed to update setting"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(Setting{Key: key, Value: req.Value, Description: req.Description})
		})

		r.Delete("/{key}", func(w http.ResponseWriter, r *http.Request) {
			key := chi.URLParam(r, "key")
			result, err := db.Exec(r.Context(), `DELETE FROM app_settings WHERE key = $1`, key)
			if err != nil {
				http.Error(w, `{"error":"failed to delete setting"}`, http.StatusInternalServerError)
				return
			}
			if result.RowsAffected() == 0 {
				http.Error(w, `{"error":"setting not found"}`, http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
		})

		r.Post("/verify-openai", func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				Provider string `json:"provider"`
				Model    string `json:"model"`
			}
			json.NewDecoder(r.Body).Decode(&req)

			apiKey := getAPIKey(db)
			if apiKey == "" {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"ok": false, "error": "No hay API key configurada",
				})
				return
			}

			model := req.Model
			if model == "" {
				model = getModel(db, "")
			}
			provider := req.Provider
			if provider == "" {
				provider = getProvider(db)
			}

			var result string
			var callErr error
			if provider == "anthropic" {
				result, callErr = callAnthropic(apiKey, model, "Sos un asistente. Respondes SOLO con 'OK'.", "Responde OK")
			} else {
				result, callErr = callOpenAICompatible(provider, apiKey, model, "Sos un asistente. Respondes SOLO con 'OK'.", "Responde OK")
			}

			if callErr != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"ok": false, "error": callErr.Error(),
				})
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"ok": true, "model": model, "model_used": model, "provider": provider, "response": result,
			})
		})
	}
}
