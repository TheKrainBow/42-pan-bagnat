package redirections

import (
	"backend/database"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

func ensureRedirectionIconDir() (string, error) {
	dir := "./assets/redirection-icons"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func detectImageExt(data []byte) string {
	ct := http.DetectContentType(data)
	switch {
	case strings.Contains(ct, "png"):
		return ".png"
	case strings.Contains(ct, "jpeg"):
		return ".jpg"
	case strings.Contains(ct, "webp"):
		return ".webp"
	case strings.Contains(ct, "gif"):
		return ".gif"
	default:
		return ".png"
	}
}

func saveRedirectionIcon(redirectionID string, data []byte, hintedName string) (string, error) {
	dir, err := ensureRedirectionIconDir()
	if err != nil {
		return "", err
	}
	ext := filepath.Ext(hintedName)
	if ext == "" || len(ext) > 5 || mime.TypeByExtension(ext) == "" {
		ext = detectImageExt(data)
	}
	dst := filepath.Join(dir, fmt.Sprintf("%s%s", redirectionID, ext))
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return "", err
	}
	return "/assets/redirection-icons/" + filepath.Base(dst), nil
}

// SetRedirectionIconUpload handles POST /admin/redirections/{redirectionID}/icon/upload (multipart/form-data: file)
func SetRedirectionIconUpload(w http.ResponseWriter, r *http.Request) {
	redirectionID := chi.URLParam(r, "redirectionID")
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file", http.StatusBadRequest)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "failed to read file", http.StatusBadRequest)
		return
	}

	url, err := saveRedirectionIcon(redirectionID, data, "upload")
	if err != nil {
		http.Error(w, "failed to save icon", http.StatusInternalServerError)
		return
	}
	if _, err := database.PatchRedirection(database.RedirectionPatch{ID: redirectionID, IconURL: &url}); err != nil {
		http.Error(w, "failed to update redirection", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"icon_url": url})
}

// SetRedirectionIconFromURL handles POST /admin/redirections/{redirectionID}/icon/url { "url": "https://..." }
func SetRedirectionIconFromURL(w http.ResponseWriter, r *http.Request) {
	redirectionID := chi.URLParam(r, "redirectionID")
	var body struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.URL) == "" {
		http.Error(w, "invalid url", http.StatusBadRequest)
		return
	}
	resp, err := http.Get(body.URL)
	if err != nil || resp.StatusCode != 200 {
		http.Error(w, "failed to download", http.StatusBadRequest)
		return
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "failed to read", http.StatusBadRequest)
		return
	}
	url, err := saveRedirectionIcon(redirectionID, data, filepath.Base(body.URL))
	if err != nil {
		http.Error(w, "failed to save icon", http.StatusInternalServerError)
		return
	}
	if _, err := database.PatchRedirection(database.RedirectionPatch{ID: redirectionID, IconURL: &url}); err != nil {
		http.Error(w, "failed to update redirection", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"icon_url": url})
}
