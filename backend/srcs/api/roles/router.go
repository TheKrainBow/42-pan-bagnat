package roles

import "github.com/go-chi/chi/v5"

func RegisterRoutes(r chi.Router) {
    r.Get("/", GetRoles)
    r.Post("/", PostRole)
    r.Get("/{roleID}", GetRole)
    r.Patch("/{roleID}", PatchRole)
    r.Delete("/{roleID}", DeleteRole)
    r.Put("/{roleID}/rules", PutRoleRules)
    r.Get("/{roleID}/rules", GetRoleRules)
    r.Post("/{roleID}/rules/validate", ValidateRoleRules)
    r.Post("/{roleID}/rules/evaluate", EvaluateRoleRules)
}
