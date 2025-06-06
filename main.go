package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/justinas/alice"
	"github.com/lib/pq"
	"github.com/xeipuuv/gojsonschema"
	"golang.org/x/crypto/bcrypt"
)

type App struct {
	DB     *sql.DB
	JWTKey []byte
}

type Credentials struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type Claims struct {
	Username string `json:"username"`
	ID       string `json:"id"`
	jwt.RegisteredClaims
}

type UserResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Token    string `json:"token"`
}
type ErrorResponse struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

type RouteResponse struct {
	Message string `json:"message"`
	ID      string `json:"id,omitempty"`
}

type Project struct {
	ID              string   `json:"id,omitempty"`
	UserID          string   `json:"user,omitempty"`
	Name            string   `json:"name,omitempty"`
	RepoURL         string   `json:"repo_url,omitempty"`
	SiteURL         string   `json:"site_url,omitempty"`
	Description     string   `json:"description,omitempty"`
	Dependencies    []string `json:"dependencies,omitempty"`
	DevDependencies []string `json:"dev_dependencies,omitempty"`
	Status          string   `json:"status,omitempty"`
}

func main() {
	err := godotenv.Load()

	if err != nil {
		log.Fatal("error loading .env file: ", err)
	}

	var loadErr error
	userSchema, loadErr := loadSchema("schemas/user.json")

	if loadErr != nil {
		log.Fatalf("Error loading user schema %v", loadErr)
	}

	projectSchema, loadErr := loadSchema("schemas/project.json")
	if loadErr != nil {
		log.Fatalf("Error loading project schema %v", loadErr)
	}

	DBNAME := os.Getenv("DBNAME")
	DBHOST := os.Getenv("DBHOST")
	DBPASSWORD := os.Getenv("DBPASSWORD")
	DBUSER := os.Getenv("DBUSER")
	DBPORT := os.Getenv("DBPORT")
	JWTKey := os.Getenv("JWT_SECRET")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		DBHOST, DBPORT, DBUSER, DBPASSWORD, DBNAME)

	DB, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer DB.Close()

	app := &App{DB: DB, JWTKey: []byte(JWTKey)}

	err = app.DB.Ping()
	if err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	fmt.Println("Successfully connected to PostgreSQL!")

	log.Println("Starting server...")

	router := mux.NewRouter()

	log.Println("Setting up routes...")

	userChain := alice.New(loggingMiddleware, validateMiddleware(userSchema))
	router.Handle("/register", userChain.ThenFunc(app.register)).Methods("POST")
	router.Handle("/login", userChain.ThenFunc(app.login)).Methods("POST")

	projectChain := alice.New(loggingMiddleware, app.jwtMiddleware)
	router.Handle("/projects", projectChain.ThenFunc(app.getProjects)).Methods("GET")
	router.Handle("/projects/{id}", projectChain.ThenFunc(app.getProject)).Methods("GET")
	router.Handle("/projects/{id}", projectChain.ThenFunc(app.deleteProject)).Methods("DELETE")

	projectChainWithValidation := projectChain.Append(validateMiddleware(projectSchema))
	router.Handle("/projects", projectChainWithValidation.ThenFunc(app.createProject)).Methods("POST")
	router.Handle("/projects/{id}", projectChainWithValidation.ThenFunc(app.updateProject)).Methods("PUT")

	log.Println("Listening on port 5000")
	log.Fatal(http.ListenAndServe(":5000", router))
}

func loadSchema(filepath string) (string, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s\n", r.RemoteAddr, r.Method, r.URL)

		next.ServeHTTP(w, r)
	})
}

func (a App) jwtMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" {
			respondWithError(w, http.StatusUnauthorized, "No authorization token provided")
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
			return a.JWTKey, nil
		})

		if err != nil {
			if err == jwt.ErrSignatureInvalid {
				respondWithError(w, http.StatusUnauthorized, "Invalid token signature")
				return
			}
			respondWithError(w, http.StatusBadRequest, "Invalid token")
			return
		}

		if !token.Valid {
			respondWithError(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		ctx := context.WithValue(r.Context(), "claims", claims)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func validateMiddleware(schema string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var body map[string]interface{}

			bodyBytes, err := io.ReadAll(r.Body)

			if err != nil {
				respondWithError(w, http.StatusBadRequest, "Invalid request payload")
				return
			}

			err = json.Unmarshal(bodyBytes, &body)

			if err != nil {
				respondWithError(w, http.StatusBadRequest, "Invalid request payload")
				return
			}

			schemaLoader := gojsonschema.NewStringLoader(schema)

			documentLoader := gojsonschema.NewGoLoader(body)

			result, err := gojsonschema.Validate(schemaLoader, documentLoader)

			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "Error validating json schema")
				return
			}

			if !result.Valid() {
				var errs []string
				for _, err := range result.Errors() {
					errs = append(errs, err.String())
				}
				respondWithError(w, http.StatusBadRequest, strings.Join(errs, ", "))
				return
			}

			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			next.ServeHTTP(w, r)
		})
	}
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{Message: message, Code: code})
}

func (a App) generateToken(username string, id string) (string, error) {
	expirationTime := time.Now().Add(5 * time.Minute)

	claims := &Claims{
		Username: username,
		ID:       id,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(a.JWTKey)

	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// register function to handle user registration
func (a App) register(w http.ResponseWriter, r *http.Request) {
	var creds Credentials

	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(creds.Password), bcrypt.DefaultCost)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error hashing password")
		return
	}

	var id string
	err = a.DB.QueryRow("INSERT INTO \"users\" (username, password) VALUES ($1, $2) RETURNING id", creds.Username, string(hashedPassword)).Scan(&id)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating user")
		return
	}

	w.Header().Set("Content-type", "application/json")
	json.NewEncoder(w).Encode(UserResponse{ID: id, Username: creds.Username})
}

// login
func (a App) login(w http.ResponseWriter, r *http.Request) {
	var creds Credentials

	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	var storedCreds Credentials
	var id string

	err = a.DB.QueryRow("SELECT id, username, password FROM \"users\" WHERE username=$1", creds.Username).Scan(&id, &storedCreds.Username, &storedCreds.Password)

	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusUnauthorized, "Invalid username or password")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Invalid request payload")
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedCreds.Password), []byte(creds.Password))

	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}

	tokenString, err := a.generateToken(creds.Username, id)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error generating token")
		return
	}

	w.Header().Set("Content-type", "application/json")
	json.NewEncoder(w).Encode(UserResponse{ID: id, Username: creds.Username, Token: tokenString})
}

// createProject
func (a App) createProject(w http.ResponseWriter, r *http.Request) {
	var project Project

	err := json.NewDecoder(r.Body).Decode(&project)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	claims := r.Context().Value("claims").(*Claims)
	userID := claims.ID

	var ID string

	err = a.DB.QueryRow("INSERT INTO projects (user_id, name, repo_url,site_url,description,dependencies,dev_dependencies,status) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id",
		userID, project.Name, project.RepoURL, project.SiteURL, project.Description, pq.Array(project.Dependencies), pq.Array(project.DevDependencies), "active").Scan(&ID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating project")
		return
	}

	project.ID = ID
	project.UserID = userID

	w.Header().Set("Content-type", "application/json")
	json.NewEncoder(w).Encode(project)
}

// updateProject
func (a App) updateProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	claims := r.Context().Value("claims").(*Claims)
	userID := claims.ID

	var project Project
	err := json.NewDecoder(r.Body).Decode(&project)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	var storedUserID string
	err = a.DB.QueryRow("SELECT user_id FROM projects WHERE id = $1", id).Scan(&storedUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Project not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Error fetching project")
		return
	}
	if storedUserID != userID {
		respondWithError(w, http.StatusForbidden, "You do not have permission to update this project")
		return
	}

	_, err = a.DB.Exec(
		`UPDATE projects SET name = $1, repo_url = $2, site_url = $3, description = $4, dependencies = $5, dev_dependencies = $6, status = $7 WHERE id = $8 AND user_id = $9`,
		project.Name,
		project.RepoURL,
		project.SiteURL,
		project.Description,
		pq.Array(project.Dependencies),
		pq.Array(project.DevDependencies),
		project.Status,
		id,
		userID,
	)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating project")
		return
	}
	// Fetch the updated project
	err = a.DB.QueryRow(`SELECT id, "user_id", name, repo_url, site_url, description, dependencies, dev_dependencies, status FROM projects WHERE id = $1 AND "user_id" = $2`, id, userID).Scan(
		&project.ID,
		&project.UserID,
		&project.Name,
		&project.RepoURL,
		&project.SiteURL,
		&project.Description,
		pq.Array(&project.Dependencies),
		pq.Array(&project.DevDependencies),
		&project.Status,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Project not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Error fetching updated project")
		return
	}
	// Convert pq.StringArray to []string
	project.Dependencies = []string(project.Dependencies)
	project.DevDependencies = []string(project.DevDependencies)

	// Return the updated project
	w.Header().Set("Content-type", "application/json")
	json.NewEncoder(w).Encode(project)
}

// getProjects
func (a App) getProjects(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value("claims").(*Claims)
	userID := claims.ID

	// Fixed: Use "user" instead of "user_id" and added "user" to SELECT
	rows, err := a.DB.Query(`SELECT id, "user_id", name, repo_url, site_url, description, dependencies, dev_dependencies, status FROM projects WHERE "user_id" = $1`, userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error fetching projects")
		return
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var project Project
		var dependencies, devDependencies pq.StringArray

		err := rows.Scan(
			&project.ID,
			&project.UserID,
			&project.Name,
			&project.RepoURL,
			&project.SiteURL,
			&project.Description,
			&dependencies,
			&devDependencies,
			&project.Status,
		)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Error scanning project")
			return
		}

		// Convert pq.StringArray to []string
		project.Dependencies = []string(dependencies)
		project.DevDependencies = []string(devDependencies)

		projects = append(projects, project)
	}

	if err := rows.Err(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error processing projects")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projects)
}

// getProject
func (a App) getProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	claims := r.Context().Value("claims").(*Claims)
	userID := claims.ID

	var project Project
	err := a.DB.QueryRow(`SELECT id, "user_id", name, repo_url, site_url, description, dependencies, dev_dependencies, status FROM projects WHERE id = $1 AND "user_id" = $2`, id, userID).Scan(
		&project.ID, &project.UserID, &project.Name, &project.RepoURL, &project.SiteURL, &project.Description, pq.Array(&project.Dependencies), pq.Array(&project.DevDependencies), &project.Status,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Project not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Convert pq.StringArray to []string
	project.Dependencies = []string(project.Dependencies)
	project.DevDependencies = []string(project.DevDependencies)

	w.Header().Set("Content-type", "application/json")
	json.NewEncoder(w).Encode(project)
}

// deleteProject
func (a App) deleteProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	claims := r.Context().Value("claims").(*Claims)
	userID := claims.ID
	var storedUserID string
	err := a.DB.QueryRow("SELECT user_id FROM projects WHERE id = $1", id).Scan(&storedUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Project not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Error fetching project")
		return
	}

	if storedUserID != userID {
		respondWithError(w, http.StatusForbidden, "You do not have permission to delete this project")
		return
	}

	_, err = a.DB.Exec("DELETE FROM projects WHERE id = $1 AND user_id = $2", id, userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error deleting project")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
