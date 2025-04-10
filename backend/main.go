package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	port := getPort()

	// Set up the CORS middleware
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"}, // allow React frontend to access the backend
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	// Use the CORS middleware
	r.Use(corsMiddleware.Handler)

	r.Get("/api", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Received request for /api")
		fmt.Fprintln(w, "Hello from Go backend with chi!")
	})

	r.Get("/api/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"version": "1.1"}`)
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
