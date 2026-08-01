package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Notification struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	Link      string `json:"link"`
	Read      bool   `json:"read"`
	CreatedAt string `json:"created_at"`
}

func CreateNotification(ctx context.Context, db *pgxpool.Pool, userID, notifType, title, message, link string) error {
	_, err := db.Exec(ctx, `
		INSERT INTO notifications (user_id, type, title, message, link)
		VALUES ($1, $2, $3, $4, $5)`,
		userID, notifType, title, message, link)
	return err
}

func NotificationsRoutes(db *pgxpool.Pool, cfg *config.Config) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			userID := r.Context().Value(UserIDKey).(string)
			onlyUnread := r.URL.Query().Get("unread") == "true"
			limit := 20
			if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 100 {
				limit = l
			}

			query := `SELECT id, type, title, message, COALESCE(link, ''), read, created_at
			          FROM notifications WHERE user_id = $1`
			args := []any{userID}
			if onlyUnread {
				query += ` AND read = FALSE`
			}
			query += ` ORDER BY created_at DESC LIMIT ` + strconv.Itoa(limit)

			rows, err := db.Query(r.Context(), query, args...)
			if err != nil {
				http.Error(w, `{"error":"failed to get notifications"}`, http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			notifications := []Notification{}
			for rows.Next() {
				var n Notification
				var created time.Time
				if err := rows.Scan(&n.ID, &n.Type, &n.Title, &n.Message, &n.Link, &n.Read, &created); err != nil {
					continue
				}
				n.CreatedAt = created.Format(time.RFC3339)
				notifications = append(notifications, n)
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(notifications)
		})

		r.Get("/unread-count", func(w http.ResponseWriter, r *http.Request) {
			userID := r.Context().Value(UserIDKey).(string)
			var count int
			if err := db.QueryRow(r.Context(),
				`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read = FALSE`, userID).Scan(&count); err != nil {
				http.Error(w, `{"error":"failed to count notifications"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]int{"count": count})
		})

		r.Put("/{notificationID}/read", func(w http.ResponseWriter, r *http.Request) {
			userID := r.Context().Value(UserIDKey).(string)
			notificationID := chi.URLParam(r, "notificationID")
			_, err := db.Exec(r.Context(),
				`UPDATE notifications SET read = TRUE WHERE id = $1 AND user_id = $2`,
				notificationID, userID)
			if err != nil {
				http.Error(w, `{"error":"failed to mark notification as read"}`, http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})

		r.Put("/read-all", func(w http.ResponseWriter, r *http.Request) {
			userID := r.Context().Value(UserIDKey).(string)
			_, err := db.Exec(r.Context(),
				`UPDATE notifications SET read = TRUE WHERE user_id = $1 AND read = FALSE`, userID)
			if err != nil {
				http.Error(w, `{"error":"failed to mark all as read"}`, http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})
	}
}
