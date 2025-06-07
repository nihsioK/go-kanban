package user

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/nihsioK/go-kanban/internal/app"
	"golang.org/x/crypto/bcrypt"
)

// Register godoc
// @Summary Register a new user
// @Description Create a new user account
// @Tags users
// @Accept json
// @Produce json
// @Param user body user.Credentials true "User credentials"
// @Success 200 {object} user.UserResponse
// @Failure 400 {object} app.ErrorResponse
// @Router /register [post]
func Register(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var creds Credentials
		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			app.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(creds.Password), 8)
		if err != nil {
			app.RespondWithError(w, http.StatusInternalServerError, "Error hashing password")
			return
		}

		var id string
		err = a.DB.QueryRow("INSERT INTO users (username, password) VALUES ($1, $2) RETURNING id",
			creds.Username, string(hashedPassword)).Scan(&id)

		if err != nil {
			app.RespondWithError(w, http.StatusInternalServerError, "Error creating user")
			return
		}

		token, err := GenerateToken(a.JWTKey, creds.Username, id)
		if err != nil {
			app.RespondWithError(w, http.StatusInternalServerError, "Error generating token")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(UserResponse{ID: id, Username: creds.Username, Token: token})
	}
}

// Login godoc
// @Summary Login a user
// @Description Authenticate a user and return a JWT token
// @Tags users
// @Accept json
// @Produce json
// @Param user body user.Credentials true "User credentials"
// @Success 200 {object} user.UserResponse
// @Failure 400 {object} app.ErrorResponse
// @Failure 401 {object} app.ErrorResponse
// @Router /login [post]
func Login(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var creds Credentials
		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			app.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}

		var storedCreds Credentials
		var id string
		err := a.DB.QueryRow("SELECT id, username, password FROM users WHERE username=$1",
			creds.Username).Scan(&id, &storedCreds.Username, &storedCreds.Password)

		if err != nil {
			if err == sql.ErrNoRows {
				app.RespondWithError(w, http.StatusUnauthorized, "Invalid username or password")
				return
			}
			app.RespondWithError(w, http.StatusInternalServerError, "Error fetching user")
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(storedCreds.Password), []byte(creds.Password)); err != nil {
			app.RespondWithError(w, http.StatusUnauthorized, "Invalid username or password")
			return
		}

		token, err := GenerateToken(a.JWTKey, creds.Username, id)
		if err != nil {
			app.RespondWithError(w, http.StatusInternalServerError, "Error generating token")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(UserResponse{ID: id, Username: creds.Username, Token: token})
	}
}
