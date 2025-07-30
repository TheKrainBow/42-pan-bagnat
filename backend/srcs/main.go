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
	"backend/websocket"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/lib/pq"
)

// @title Pan Bagnat API
// @version 1.1
// @description API REST du projet Pan Bagnat.
// @host heinz.42nice.fr:8080
// @BasePath /api/v1
func main() {
	port := getPort()

	// Set up the CORS middleware
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins: []string{
			"http://localhost:*",
			"http://localhost",
			"https://localhost:*",
			"https://localhost",
			"http://127.0.0.1:*",
			"http://127.0.0.1",
			"https://127.0.0.1:*",
			"https://127.0.0.1",
			"http://heinz.42nice.fr",
			"https://heinz.42nice.fr",
		},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(corsMiddleware.Handler)

	r.Get("/swagger/*", httpSwagger.WrapHandler)

	r.Get("/module-page/{pageName}", modules.PageRedirection)
	r.Get("/module-page/{pageName}/*", modules.PageRedirection)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/modules", func(r chi.Router) {
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

	go websocket.Dispatch()

	// Mount WebSocket endpoint
	r.HandleFunc("/ws", websocket.Handler())

	// Webhook endpoint pushes into websocket.Events
	r.Post("/webhooks/events", websocket.WebhookHandler(websocket.Secret))
	log.Printf("Backend listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func getPort() string {
	if port := os.Getenv("PORT"); port != "" {
		return port
	}
	return "8080"
}
