package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func LogAuditEvent(db *pgxpool.Pool, entityType, entityID, action, userID string, oldValue, newValue interface{}) error {
	var oldJSON, newJSON []byte
	var err error

	if oldValue != nil {
		oldJSON, err = json.Marshal(oldValue)
		if err != nil {
			return err
		}
	}
	if newValue != nil {
		newJSON, err = json.Marshal(newValue)
		if err != nil {
			return err
		}
	}

	var userIDPtr *string
	if userID != "" {
		userIDPtr = &userID
	}

	_, err = db.Exec(context.Background(),
		`INSERT INTO audit_log_entries (entity_type, entity_id, action, user_id, old_value, new_value)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		entityType, entityID, action, userIDPtr, oldJSON, newJSON)
	return err
}

func GetAuditLog(db *pgxpool.Pool, entityType, entityID string) ([]map[string]interface{}, error) {
	rows, err := db.Query(context.Background(),
		`SELECT id, entity_type, entity_id, action, user_id, old_value, new_value, version, ip_address, user_agent, created_at
		 FROM audit_log_entries
		 WHERE entity_type = $1 AND entity_id = $2
		 ORDER BY created_at DESC`,
		entityType, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []map[string]interface{}
	for rows.Next() {
		var id, et, eid, action string
		var userID, ipAddr, userAgent *string
		var oldValue, newValue json.RawMessage
		var version *int
		var createdAt interface{}

		if err := rows.Scan(&id, &et, &eid, &action, &userID, &oldValue, &newValue, &version, &ipAddr, &userAgent, &createdAt); err != nil {
			continue
		}

		entry := map[string]interface{}{
			"id":          id,
			"entity_type": et,
			"entity_id":   eid,
			"action":      action,
			"created_at":  createdAt,
		}
		if userID != nil {
			entry["user_id"] = *userID
		}
		if oldValue != nil {
			entry["old_value"] = oldValue
		}
		if newValue != nil {
			entry["new_value"] = newValue
		}
		if version != nil {
			entry["version"] = *version
		}
		if ipAddr != nil {
			entry["ip_address"] = *ipAddr
		}
		if userAgent != nil {
			entry["user_agent"] = *userAgent
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func AuditRoutes(db *pgxpool.Pool) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/{entityType}/{entityID}", func(w http.ResponseWriter, r *http.Request) {
			entityType := chi.URLParam(r, "entityType")
			entityID := chi.URLParam(r, "entityID")

			entries, err := GetAuditLog(db, entityType, entityID)
			if err != nil {
				http.Error(w, `{"error":"failed to get audit log"}`, http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(entries)
		})
	}
}
