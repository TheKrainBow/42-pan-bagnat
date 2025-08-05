package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"backend/api/auth"
	"backend/api/modules"
	"backend/api/ping"
	"backend/api/roles"
	"backend/api/users"
	_ "backend/docs"
	"backend/utils"
	"backend/websocket"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"

	_ "github.com/lib/pq"
)

// @title Pan Bagnat API
// @version 1.1
// @description API REST du projet Pan Bagnat.
// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey SessionAuth
// @in cookie
// @name session_id

// @securityDefinitions.apikey AdminAuth
// @in cookie
// @name session_id

// @tag.name      Users
// @tag.description Operations for managing user accounts, profiles, and permissions

// @tag.name      Roles
// @tag.description Endpoints for creating, updating, and deleting roles and their assignments

// @tag.name      Pages
// @tag.description Module front-end page configuration, management, and proxy routing

// @tag.name      Docker
// @tag.description Module container lifecycle operations (start, stop, restart, logs, delete)

// @tag.name      Git
// @tag.description Module source repository operations (clone, pull, update remote)

// @tag.name      Modules
// @tag.description Core module lifecycle operations: import, list, update, and delete

func main() {
	port := getPort()

	if os.Getenv("BUILD_MODE") == "" {
		err := godotenv.Load("../.env")
		if err != nil {
			log.Println("No .env file found, and BUILD_MODE not set! (may be fine in production)")
		}
	}
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

	r.Get("/api/swagger-public.json", func(w http.ResponseWriter, r *http.Request) {
		raw, err := utils.LoadRawSpec()
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to load spec %s", err.Error()), 500)
			return
		}
		pub := utils.FilterSpec(raw, func(p string) bool {
			return !strings.HasPrefix(p, "/admin/")
		})
		utils.PruneTags(pub)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pub)
	})

	r.Get("/api/swagger-admin.json", func(w http.ResponseWriter, r *http.Request) {
		raw, err := utils.LoadRawSpec()
		if err != nil {
			http.Error(w, "failed to load spec", 500)
			return
		}
		adm := utils.FilterSpec(raw, func(p string) bool {
			return strings.HasPrefix(p, "/admin/")
		})
		utils.PruneTags(adm)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(adm)
	})

	r.Get("/api/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		raw, err := utils.LoadRawSpec()
		if err != nil {
			http.Error(w, "failed to load spec", 500)
			return
		}
		utils.PruneTags(raw)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(raw)
	})

	fs := http.FileServer(http.Dir("./docs/swagger-ui"))
	r.Handle("/api/v1/docs/*", http.StripPrefix("/api/v1/docs/", fs))

	// r.Get("/swagger/*", httpSwagger.WrapHandler)

	r.Route("/auth", func(r chi.Router) {
		auth.RegisterRoutes(r)
	})

	r.With(auth.PageAccessMiddleware).Get("/module-page/{pageName}", modules.PageRedirection)
	r.With(auth.PageAccessMiddleware).Get("/module-page/{pageName}/*", modules.PageRedirection)

	r.With(auth.AuthMiddleware).Get("/api/v1/users/me", users.GetUserMe)
	r.With(auth.AuthMiddleware).Get("/api/v1/users/me/pages", modules.GetPages)
	r.With(auth.AuthMiddleware).Get("/api/v1/ping", ping.Ping)

	r.Route("/api/v1/admin", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(auth.AuthMiddleware, auth.AdminMiddleware)

			r.Route("/modules", modules.RegisterRoutes)
			r.Route("/users", users.RegisterRoutes)
			r.Route("/roles", roles.RegisterRoutes)
		})
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
