package main

import (
	"log"
	"net/http"
	"os"

	"backend/api/modules"
	"backend/api/roles"
	"backend/api/users"
	"backend/api/version"
	_ "backend/docs"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/lib/pq"
)

// @title Pan Bagnat API
// @version 1.1
// @description API REST du projet Pan Bagnat.
// @host localhost:8080
// @BasePath /api/v1
func main() {
	port := getPort()

	// Set up the CORS middleware
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost*"}, // allow React frontend to access the backend
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	// Use the CORS middleware
	r.Use(corsMiddleware.Handler)
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/modules", func(r chi.Router) {
			// We must register routes in the package the have the handlers, so chi doesn't break
			modules.RegisterRoutes(r)
		})
		r.Route("/users", func(r chi.Router) {
			users.RegisterRoutes(r)
		})
		r.Route("/roles", func(r chi.Router) {
			roles.RegisterRoutes(r)
		})
		r.Get("/version", version.GetVersion)
	})

	log.Printf("Backend listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func getPort() string {
	if port := os.Getenv("PORT"); port != "" {
		return port
	}
	return "8080"
}
