package modules

import (
	"backend/core"
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
	"log"
)

func ensureIconDir() (string, error) {
	dir := "./assets/module-icons"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func ensurePageIconDir() (string, error) {
	dir := "./assets/page-icons"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func detectImageExt(data []byte) string {
	ct := http.DetectContentType(data)
	switch ct {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		// try by sniffing from header
		if strings.Contains(ct, "png") {
			return ".png"
		}
		if strings.Contains(ct, "jpeg") {
			return ".jpg"
		}
		if strings.Contains(ct, "webp") {
			return ".webp"
		}
		if strings.Contains(ct, "gif") {
			return ".gif"
		}
		return ".png"
	}
}

func saveModuleIcon(moduleID string, data []byte, hintedName string) (string, error) {
	dir, err := ensureIconDir()
	if err != nil {
		return "", err
	}
	ext := filepath.Ext(hintedName)
	if ext == "" || len(ext) > 5 {
		ext = detectImageExt(data)
		if ext == "" {
			ext = ".png"
		}
	}
	// Normalize ext
	if mt := mime.TypeByExtension(ext); mt == "" {
		ext = detectImageExt(data)
	}
	dst := filepath.Join(dir, fmt.Sprintf("%s%s", moduleID, ext))
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return "", err
	}
	// public URL
	return "/assets/module-icons/" + filepath.Base(dst), nil
}

func savePageIcon(pageID string, data []byte, hintedName string) (string, error) {
	dir, err := ensurePageIconDir()
	if err != nil {
		return "", err
	}
	ext := filepath.Ext(hintedName)
	if ext == "" || len(ext) > 5 {
		ext = detectImageExt(data)
		if ext == "" {
			ext = ".png"
		}
	}
	if mt := mime.TypeByExtension(ext); mt == "" {
		ext = detectImageExt(data)
	}
	dst := filepath.Join(dir, fmt.Sprintf("%s%s", pageID, ext))
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return "", err
	}
	return "/assets/page-icons/" + filepath.Base(dst), nil
}

// POST /admin/modules/{moduleID}/icon/upload (multipart/form-data: file)
func SetModuleIconUpload(w http.ResponseWriter, r *http.Request) {
	moduleID := chi.URLParam(r, "moduleID")
	if moduleID == "" {
		http.Error(w, "missing moduleID", http.StatusBadRequest)
		return
	}

	file, hdr, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file", http.StatusBadRequest)
		return
	}
	defer file.Close()
	// Read full content (no artificial cap here; reverse proxies may still enforce limits)
	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "failed to read file", http.StatusBadRequest)
		return
	}

	url, err := saveModuleIcon(moduleID, data, hdr.Filename)
	if err != nil {
		http.Error(w, "failed to save icon", http.StatusInternalServerError)
		return
	}

	// Patch DB
	if _, err := core.PatchModule(core.ModulePatch{ID: moduleID, IconURL: &url}); err != nil {
		http.Error(w, "failed to update module", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"icon_url": url})
}

// POST /admin/modules/{moduleID}/icon/url { "url": "https://..." }
func SetModuleIconFromURL(w http.ResponseWriter, r *http.Request) {
	moduleID := chi.URLParam(r, "moduleID")
	if moduleID == "" {
		http.Error(w, "missing moduleID", http.StatusBadRequest)
		return
	}
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
		http.Error(w, "failed to read image", http.StatusBadRequest)
		return
	}
	url, err := saveModuleIcon(moduleID, data, filepath.Base(body.URL))
	if err != nil {
		http.Error(w, "failed to save icon", http.StatusInternalServerError)
		return
	}
	if _, err := core.PatchModule(core.ModulePatch{ID: moduleID, IconURL: &url}); err != nil {
		http.Error(w, "failed to update module", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"icon_url": url})
}

// POST /admin/modules/{moduleID}/icon/from-repo { "path": "relative/path/in/repo.png" }
func SetModuleIconFromRepo(w http.ResponseWriter, r *http.Request) {
	moduleID := chi.URLParam(r, "moduleID")
	if moduleID == "" {
		http.Error(w, "missing moduleID", http.StatusBadRequest)
		return
	}
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Path) == "" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	dbm, err := database.GetModule(moduleID)
	if err != nil {
		http.Error(w, "module not found", http.StatusNotFound)
		return
	}
	m := core.DatabaseModuleToModule(dbm)
	data, err := core.ReadModuleFile(m, body.Path)
	if err != nil {
		http.Error(w, "failed to read file from repo", http.StatusBadRequest)
		return
	}
	url, err := saveModuleIcon(moduleID, data, filepath.Base(body.Path))
	if err != nil {
		http.Error(w, "failed to save icon", http.StatusInternalServerError)
		return
	}
	if _, err := core.PatchModule(core.ModulePatch{ID: moduleID, IconURL: &url}); err != nil {
		http.Error(w, "failed to update module", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"icon_url": url})
}

// POST /admin/modules/{moduleID}/pages/{pageID}/icon/upload
func SetPageIconUpload(w http.ResponseWriter, r *http.Request) {
	pageID := chi.URLParam(r, "pageID")
	moduleID := chi.URLParam(r, "moduleID")
	log.Printf("[page-icon][upload] moduleID=%s pageID=%s contentLength=%d", moduleID, pageID, r.ContentLength)
	file, _, err := r.FormFile("file")
	if err != nil {
		log.Printf("page icon upload: missing file: %v", err)
		http.Error(w, "missing file", http.StatusBadRequest)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		log.Printf("page icon upload: read error: %v", err)
		http.Error(w, "failed to read", http.StatusBadRequest)
		return
	}
	url, err := savePageIcon(pageID, data, "upload")
	if err != nil {
		log.Printf("page icon upload: save error: %v", err)
		http.Error(w, "failed to save", http.StatusInternalServerError)
		return
	}
	if _, err := database.PatchModulePage(database.ModulePagePatch{ID: pageID, IconURL: &url}); err != nil {
		log.Printf("page icon upload: DB patch error: %v", err)
		http.Error(w, "failed to update page", http.StatusInternalServerError)
		return
	}
	log.Printf("[page-icon][upload] OK moduleID=%s pageID=%s icon_url=%s (size=%d bytes)", moduleID, pageID, url, len(data))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"icon_url": url})
}

// POST /admin/modules/{moduleID}/pages/{pageID}/icon/url
func SetPageIconFromURL(w http.ResponseWriter, r *http.Request) {
	pageID := chi.URLParam(r, "pageID")
	moduleID := chi.URLParam(r, "moduleID")
	var body struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.URL) == "" {
		log.Printf("page icon url: invalid body: %v", err)
		http.Error(w, "invalid url", http.StatusBadRequest)
		return
	}
	log.Printf("[page-icon][url] moduleID=%s pageID=%s url=%s", moduleID, pageID, body.URL)
	resp, err := http.Get(body.URL)
	if err != nil || resp.StatusCode != 200 {
		status := ""
		if resp != nil {
			status = resp.Status
		}
		log.Printf("page icon url: download failed: err=%v status=%s", err, status)
		http.Error(w, "failed to download", http.StatusBadRequest)
		return
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("page icon url: read body failed: %v", err)
		http.Error(w, "failed to read", http.StatusBadRequest)
		return
	}
	url, err := savePageIcon(pageID, data, filepath.Base(body.URL))
	if err != nil {
		log.Printf("page icon url: save error: %v", err)
		http.Error(w, "failed to save", http.StatusInternalServerError)
		return
	}
	if _, err := database.PatchModulePage(database.ModulePagePatch{ID: pageID, IconURL: &url}); err != nil {
		log.Printf("page icon url: DB patch error: %v", err)
		http.Error(w, "failed to update page", http.StatusInternalServerError)
		return
	}
	log.Printf("[page-icon][url] OK moduleID=%s pageID=%s icon_url=%s", moduleID, pageID, url)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"icon_url": url})
}

// POST /admin/modules/{moduleID}/pages/{pageID}/icon/from-repo
func SetPageIconFromRepo(w http.ResponseWriter, r *http.Request) {
	moduleID := chi.URLParam(r, "moduleID")
	pageID := chi.URLParam(r, "pageID")
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Path) == "" {
		log.Printf("page icon repo: invalid body: %v", err)
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	log.Printf("[page-icon][repo] moduleID=%s pageID=%s path=%s", moduleID, pageID, body.Path)
	dbm, err := database.GetModule(moduleID)
	if err != nil {
		log.Printf("page icon repo: module not found: %v", err)
		http.Error(w, "module not found", http.StatusNotFound)
		return
	}
	m := core.DatabaseModuleToModule(dbm)
	data, err := core.ReadModuleFile(m, body.Path)
	if err != nil {
		log.Printf("page icon repo: read from repo failed: %v", err)
		http.Error(w, "failed to read from repo", http.StatusBadRequest)
		return
	}
	url, err := savePageIcon(pageID, data, filepath.Base(body.Path))
	if err != nil {
		log.Printf("page icon repo: save error: %v", err)
		http.Error(w, "failed to save", http.StatusInternalServerError)
		return
	}
	if _, err := database.PatchModulePage(database.ModulePagePatch{ID: pageID, IconURL: &url}); err != nil {
		log.Printf("page icon repo: DB patch error: %v", err)
		http.Error(w, "failed to update page", http.StatusInternalServerError)
		return
	}
	log.Printf("[page-icon][repo] OK moduleID=%s pageID=%s icon_url=%s", moduleID, pageID, url)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"icon_url": url})
}

// POST /admin/modules/{moduleID}/pages/{pageID}/icon/clear
func SetPageIconClear(w http.ResponseWriter, r *http.Request) {
	pageID := chi.URLParam(r, "pageID")
	moduleID := chi.URLParam(r, "moduleID")
	log.Printf("[page-icon][clear] moduleID=%s pageID=%s", moduleID, pageID)
	if strings.TrimSpace(pageID) == "" {
		http.Error(w, "missing pageID", http.StatusBadRequest)
		return
	}
	if err := database.SetPageIconURL(pageID, nil); err != nil {
		log.Printf("page icon clear: DB error: %v", err)
		http.Error(w, "failed to clear icon", http.StatusInternalServerError)
		return
	}
	log.Printf("[page-icon][clear] OK moduleID=%s pageID=%s", moduleID, pageID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
