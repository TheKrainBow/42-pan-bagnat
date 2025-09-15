package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"backend/api/auth"
	"backend/api/integrations"
	"backend/api/modules"
	"backend/api/ping"
	"backend/api/roles"
	"backend/api/users"
	"backend/core"
	"backend/database"
	_ "backend/docs"
	"backend/utils"
	"backend/websocket"

	apiManager "github.com/TheKrainBow/go-api"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"

	_ "github.com/lib/pq"
)

// @title Pan Bagnat API
// @version 1.1
// @description API REST du projet Pan Bagnat.
// @host {{HOST_PLACEHOLDER}}
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

func InjectUserInMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_id")
		sid := ""
		if err == nil && cookie.Value != "" {
			sid = cookie.Value
		} else {
			if hdr := r.Header.Get("session_id"); hdr != "" {
				sid = hdr
			}
		}

		if sid == "" {
			log.Println("[auth] no session_id:", err)
			next.ServeHTTP(w, r)
			return
		}

		session, err := database.GetSession(sid)
		if err != nil {
			log.Printf("[auth] failed to get session: %v", err)
			next.ServeHTTP(w, r)
			return
		}
		if session.ExpiresAt.Before(time.Now()) {
			log.Println("[auth] session expired")
			go database.PurgeExpiredSessions()
			next.ServeHTTP(w, r)
			return
		}

		user, err := core.GetUser(session.Login)
		if err != nil {
			log.Println("[auth] user not found for session:", err)
			next.ServeHTTP(w, r)
			return
		}
		log.Printf("[auth] user %s authenticated via session", user.FtLogin)

		if time.Since(user.LastSeen) > time.Minute {
			go core.TouchUserLastSeen(user.ID)
			go core.TouchSession(r.Context(), session.SessionID)
		}

		ctx := context.WithValue(r.Context(), auth.UserCtxKey, &user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func InjectPageNameMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageName := chi.URLParam(r, "pageName")
		fmt.Printf("pageName: %s\n", pageName)
		ctx := context.WithValue(r.Context(), auth.PageCtxKey, pageName)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

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
			fmt.Sprintf("http://%s", os.Getenv("HOST_NAME")),
			fmt.Sprintf("https://%s", os.Getenv("HOST_NAME")),
		},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	APIClient, err := apiManager.NewAPIClient("42", apiManager.APIClientInput{
		AuthType:     apiManager.AuthTypeClientCredentials,
		TokenURL:     "https://api.intra.42.fr/oauth/token",
		Endpoint:     "https://api.intra.42.fr/v2",
		TestPath:     "/campus/41",
		ClientID:     os.Getenv("FT_CLIENT_ID"),
		ClientSecret: os.Getenv("FT_CLIENT_SECRET"),
		Scope:        "public",
	})
	if err != nil {
		log.Panic("Failed to connect to 42API: %w", err)
	}
	err = APIClient.TestConnection()
	if err != nil {
		log.Panic("42API connection failed! %w", err)
	}

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
	r.With(InjectUserInMiddleware, InjectPageNameMiddleware, auth.BlackListMiddleware, auth.PageAccessMiddleware).Get("/module-page/{pageName}", modules.PageRedirection)
	r.With(InjectUserInMiddleware, InjectPageNameMiddleware, auth.BlackListMiddleware, auth.PageAccessMiddleware).Get("/module-page/{pageName}/*", modules.PageRedirection)

	r.With(InjectUserInMiddleware, auth.AuthMiddleware, auth.BlackListMiddleware).Get("/api/v1/users/me", users.GetUserMe)
	r.With(InjectUserInMiddleware, auth.AuthMiddleware, auth.BlackListMiddleware).Get("/api/v1/users/me/pages", users.GetContextUserPages)
	r.With(InjectUserInMiddleware, auth.AuthMiddleware, auth.BlackListMiddleware).Get("/api/v1/ping", ping.Ping)

	r.Route("/api/v1/admin", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(InjectUserInMiddleware, auth.AuthMiddleware, auth.BlackListMiddleware, auth.AdminMiddleware)

			r.Route("/integrations", integrations.RegisterRoutes)
			r.Route("/modules", modules.RegisterRoutes)
			r.Get("/docker/ls", modules.GetAllContainers)
			r.Route("/users", users.RegisterRoutes)
			r.Route("/roles", roles.RegisterRoutes)
		})
	})

	go websocket.Dispatch()

	// Wire WS subscribe/unsubscribe hooks for container log streaming
	websocket.OnSubscribe = core.WSOnSubscribe
	websocket.OnUnsubscribe = core.WSOnUnsubscribe

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
