package routes

import (
	"net/http"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/nihsioK/go-kanban/internal/app"
	"github.com/nihsioK/go-kanban/internal/project"
	"github.com/nihsioK/go-kanban/internal/user"
)

func SetupRouter(a *app.App) *mux.Router {
	r := mux.NewRouter()

	// Swagger docs
	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	// Public routes
	r.Handle("/register", a.Logging(a.Validate("user", user.Register(a)))).Methods("POST")
	r.Handle("/login", a.Logging(user.Login(a))).Methods("POST")

	// Protected routes
	projectRouter := r.PathPrefix("/projects").Subrouter()
	projectRouter.Use(a.Logging)
	projectRouter.Use(a.JWTAuth)

	projectRouter.Handle("", http.HandlerFunc(project.GetAll(a))).Methods("GET")
	projectRouter.Handle("/{id}", http.HandlerFunc(project.GetOne(a))).Methods("GET")
	projectRouter.Handle("/{id}", http.HandlerFunc(project.Delete(a))).Methods("DELETE")
	projectRouter.Handle("", a.Validate("project", project.Create(a))).Methods("POST")
	projectRouter.Handle("/{id}", a.Validate("project", project.Update(a))).Methods("PUT")

	return r
}
