package dashboard

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func ensureDashboardImageDir() (string, error) {
	dir := "./assets/dashboard-images"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func detectDashboardImageExt(data []byte) string {
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

func randomFileID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func saveDashboardImage(data []byte, hintedName string) (string, error) {
	dir, err := ensureDashboardImageDir()
	if err != nil {
		return "", err
	}
	ext := filepath.Ext(hintedName)
	if ext == "" || len(ext) > 5 || mime.TypeByExtension(ext) == "" {
		ext = detectDashboardImageExt(data)
	}
	id, err := randomFileID()
	if err != nil {
		return "", err
	}
	dst := filepath.Join(dir, fmt.Sprintf("%s%s", id, ext))
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return "", err
	}
	return "/assets/dashboard-images/" + filepath.Base(dst), nil
}

// UploadDashboardImage saves an uploaded image/gif for use on the dashboard canvas.
// @Security     SessionAuth
// @Summary      Upload Dashboard Image
// @Description  Uploads an image or GIF file and returns its public URL for use on the dashboard canvas.
// @Tags         Dashboard
// @Accept       multipart/form-data
// @Produce      json
// @Param        file  formData  file  true  "Image or GIF file"
// @Success      200   {object}  map[string]string
// @Failure      400   {string}  string  "missing file"
// @Failure      500   {string}  string  "failed to save file"
// @Router       /admin/dashboard/image/upload [post]
func UploadDashboardImage(w http.ResponseWriter, r *http.Request) {
	file, hdr, err := r.FormFile("file")
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

	url, err := saveDashboardImage(data, hdr.Filename)
	if err != nil {
		http.Error(w, "failed to save file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": url})
}
