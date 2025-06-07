// @title           Test
// @version         3.0
// @description     testing
// (omit @host)
// @BasePath        /
// @schemes         http
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

package main

import (
	"log"
	"net/http"

	_ "github.com/nihsioK/go-kanban/docs"
	"github.com/nihsioK/go-kanban/internal/app"
	"github.com/nihsioK/go-kanban/internal/routes"
)

func main() {
	a := app.Initialize()
	router := routes.SetupRouter(a)

	log.Println("Server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
