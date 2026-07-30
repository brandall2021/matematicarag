package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func StudentRoutes(db *pgxpool.Pool) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/my-progress", func(w http.ResponseWriter, r *http.Request) {
			studentID := r.Context().Value(UserIDKey).(string)
			profile, err := GetOrCreateProfile(db, studentID, "matematica-1")
			if err != nil {
				http.Error(w, `{"error":"failed to load profile"}`, http.StatusInternalServerError)
				return
			}
			mastery, err := GetMasteryMap(db, studentID, "matematica-1")
			if err != nil {
				http.Error(w, `{"error":"failed to load mastery"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"profile": profile,
				"mastery": mastery,
			})
		})

		r.Get("/recommendations", func(w http.ResponseWriter, r *http.Request) {
			studentID := r.Context().Value(UserIDKey).(string)
			errors, err := GetStudentErrors(db, studentID)
			if err != nil {
				http.Error(w, `{"error":"failed to load data"}`, http.StatusInternalServerError)
				return
			}
			weakTopics := make([]string, 0)
			for _, e := range errors {
				if e.Count > 2 {
					weakTopics = append(weakTopics, e.ConceptID)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"recommendations": weakTopics,
			})
		})
	}
}
