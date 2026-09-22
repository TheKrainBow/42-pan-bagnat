package redirections

import (
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router) {
	r.Get("/", GetRedirections)
	r.Post("/", PostRedirection)

	r.Get("/{redirectionID}", GetRedirection)
	r.Patch("/{redirectionID}", PatchRedirection)
	r.Delete("/{redirectionID}", DeleteRedirection)

	r.Post("/{redirectionID}/roles/{roleID}", PostRedirectionRole)
	r.Delete("/{redirectionID}/roles/{roleID}", DeleteRedirectionRole)
	r.Post("/{redirectionID}/forbidden-roles/{roleID}", PostRedirectionForbiddenRole)
	r.Delete("/{redirectionID}/forbidden-roles/{roleID}", DeleteRedirectionForbiddenRole)

	r.Post("/{redirectionID}/icon/upload", SetRedirectionIconUpload)
	r.Post("/{redirectionID}/icon/url", SetRedirectionIconFromURL)
}
