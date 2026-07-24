package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	LastName string `json:"lastName"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
	User         User   `json:"user"`
}

type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	LastName string `json:"lastName"`
	Role     string `json:"role"`
}

func AuthRoutes(db *pgxpool.Pool, cfg *config.Config) func(r chi.Router) {
	return func(r chi.Router) {
		r.Post("/register", func(w http.ResponseWriter, r *http.Request) {
			var req RegisterRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
				return
			}
			if req.Email == "" || req.Password == "" || req.Name == "" {
				http.Error(w, `{"error":"email, password and name are required"}`, http.StatusBadRequest)
				return
			}
			hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
			if err != nil {
				http.Error(w, `{"error":"failed to hash password"}`, http.StatusInternalServerError)
				return
			}
			var user User
			err = db.QueryRow(r.Context(),
				`INSERT INTO users (email, password_hash, name, last_name, role)
				 VALUES ($1, $2, $3, $4, 'STUDENT')
				 RETURNING id, email, name, COALESCE(last_name, ''), role`,
				req.Email, string(hash), req.Name, req.LastName,
			).Scan(&user.ID, &user.Email, &user.Name, &user.LastName, &user.Role)
			if err != nil {
				if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
					http.Error(w, `{"error":"email already exists"}`, http.StatusConflict)
					return
				}
				http.Error(w, `{"error":"failed to create user"}`, http.StatusInternalServerError)
				return
			}
			token, refreshToken, err := generateTokens(user, cfg.JWTSecret)
			if err != nil {
				http.Error(w, `{"error":"failed to generate tokens"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(AuthResponse{Token: token, RefreshToken: refreshToken, User: user})
		})

		r.Post("/login", func(w http.ResponseWriter, r *http.Request) {
			var req LoginRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
				return
			}
			var user User
			var passwordHash string
			err := db.QueryRow(r.Context(),
				`SELECT id, email, name, COALESCE(last_name, ''), role, password_hash
				 FROM users WHERE email = $1`, req.Email,
			).Scan(&user.ID, &user.Email, &user.Name, &user.LastName, &user.Role, &passwordHash)
			if err != nil {
				http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
				return
			}
			if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
				http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
				return
			}
			token, refreshToken, err := generateTokens(user, cfg.JWTSecret)
			if err != nil {
				http.Error(w, `{"error":"failed to generate tokens"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(AuthResponse{Token: token, RefreshToken: refreshToken, User: user})
		})

		r.Post("/refresh", func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				RefreshToken string `json:"refreshToken"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
				return
			}
			claims, err := extractJWT(req.RefreshToken, cfg.JWTSecret)
			if err != nil {
				http.Error(w, `{"error":"invalid refresh token"}`, http.StatusUnauthorized)
				return
			}
			user := User{
				ID:    claims["sub"].(string),
				Email: claims["email"].(string),
				Name:  claims["name"].(string),
				Role:  claims["role"].(string),
			}
			token, refreshToken, err := generateTokens(user, cfg.JWTSecret)
			if err != nil {
				http.Error(w, `{"error":"failed to generate tokens"}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(AuthResponse{Token: token, RefreshToken: refreshToken, User: user})
		})
	}
}

func generateTokens(user User, secret string) (string, string, error) {
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID, "email": user.Email, "name": user.Name, "role": user.Role,
		"exp": time.Now().Add(15 * time.Minute).Unix(), "iat": time.Now().Unix(),
	})
	accessStr, err := accessToken.SignedString([]byte(secret))
	if err != nil {
		return "", "", err
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID, "email": user.Email, "name": user.Name, "role": user.Role,
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(), "iat": time.Now().Unix(),
	})
	refreshStr, err := refreshToken.SignedString([]byte(secret))
	if err != nil {
		return "", "", err
	}
	return accessStr, refreshStr, nil
}

func extractJWT(tokenString, secret string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, jwt.ErrTokenInvalidClaims
}
