package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UpdateRoleRequest struct {
	Role string `json:"role"`
}

func UserRoutes(db *pgxpool.Pool) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			rows, err := db.Query(r.Context(),
				`SELECT id, email, name, COALESCE(last_name, ''), role, created_at
				 FROM users ORDER BY created_at DESC`)
			if err != nil {
				http.Error(w, `{"error":"failed to list users"}`, http.StatusInternalServerError)
				return
			}
			defer rows.Close()
			type UserWithDate struct {
				ID        string `json:"id"`
				Email     string `json:"email"`
				Name      string `json:"name"`
				LastName  string `json:"lastName"`
				Role      string `json:"role"`
				CreatedAt string `json:"createdAt"`
			}
			users := make([]UserWithDate, 0)
			for rows.Next() {
				var u UserWithDate
				if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.LastName, &u.Role, &u.CreatedAt); err == nil {
					users = append(users, u)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(users)
		})

		r.Put("/{id}/role", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")
			var req UpdateRoleRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
				return
			}
			if req.Role != "ADMIN" && req.Role != "TEACHER" && req.Role != "STUDENT" {
				http.Error(w, `{"error":"invalid role: must be ADMIN, TEACHER, or STUDENT"}`, http.StatusBadRequest)
				return
			}
			result, err := db.Exec(r.Context(),
				`UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2`, req.Role, id)
			if err != nil {
				http.Error(w, `{"error":"failed to update role"}`, http.StatusInternalServerError)
				return
			}
			if result.RowsAffected() == 0 {
				http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "updated", "role": req.Role})
		})

		r.Delete("/{id}", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")
			result, err := db.Exec(r.Context(), `DELETE FROM users WHERE id = $1`, id)
			if err != nil {
				http.Error(w, `{"error":"failed to delete user"}`, http.StatusInternalServerError)
				return
			}
			if result.RowsAffected() == 0 {
				http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
		})
	}
}
