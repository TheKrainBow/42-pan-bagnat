package oidc

import "github.com/go-chi/chi/v5"

func RegisterPublicRoutes(r chi.Router) {
	r.Get("/.well-known/openid-configuration", GetDiscovery)
	r.Get("/.well-known/jwks.json", GetJWKS)
	r.Get("/oauth/authorize", Authorize)
	r.Post("/oauth/token", Token)
	r.Get("/oauth/userinfo", UserInfo)
}

func RegisterAdminRoutes(r chi.Router) {
	r.Get("/{moduleID}/oidc", GetModuleOIDC)
	r.Patch("/{moduleID}/oidc", PatchModuleOIDC)
	r.Post("/{moduleID}/oidc/secret", GenerateModuleOIDCSecret)
	r.Post("/{moduleID}/oidc/secret/rotate", RotateModuleOIDCSecret)
	r.Post("/{moduleID}/oidc/redirect-uris", AddModuleOIDCRedirectURI)
	r.Delete("/{moduleID}/oidc/redirect-uris", DeleteModuleOIDCRedirectURI)
}
