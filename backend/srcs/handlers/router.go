package handlers

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

var r *chi.Mux

func NewRouter() *chi.Mux {
	r = chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(cors.AllowAll().Handler) // or your CORS setup
	return r
}

func GetRouter() *chi.Mux {
	if r == nil {
		return NewRouter()
	}
	return r
}
