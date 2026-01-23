package api

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FSHandler handles file system operations
type FSHandler struct {
	workDir   string
	authToken string
}

// NewFSHandler creates a new file system handler
func NewFSHandler(workDir, authToken string) *FSHandler {
	return &FSHandler{
		workDir:   workDir,
		authToken: authToken,
	}
}

// FileInfo represents file metadata
type FileInfo struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	IsDir   bool   `json:"is_dir"`
	Size    int64  `json:"size"`
	ModTime string `json:"mod_time"`
}

// ServeHTTP implements http.Handler
func (h *FSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	token := r.Header.Get("Authorization")
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	if strings.TrimPrefix(token, "Bearer ") != h.authToken {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract path from URL (remove /api/fs prefix)
	reqPath := strings.TrimPrefix(r.URL.Path, "/api/fs")
	if reqPath == "" {
		reqPath = "/"
	}

	// Resolve and validate path
	fullPath, err := h.resolvePath(reqPath)
	if err != nil {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r, fullPath, reqPath)
	case http.MethodPut:
		h.handlePut(w, r, fullPath)
	case http.MethodDelete:
		h.handleDelete(w, r, fullPath)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// resolvePath validates and resolves a path to prevent path traversal
func (h *FSHandler) resolvePath(reqPath string) (string, error) {
	// Clean the path
	cleanPath := filepath.Clean(reqPath)

	// Join with work directory
	fullPath := filepath.Join(h.workDir, cleanPath)

	// Ensure the path is within work directory
	absWorkDir, err := filepath.Abs(h.workDir)
	if err != nil {
		return "", err
	}
	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(absFullPath, absWorkDir) {
		return "", os.ErrPermission
	}

	return fullPath, nil
}

// handleGet handles GET requests - read file or list directory
func (h *FSHandler) handleGet(w http.ResponseWriter, r *http.Request, fullPath, reqPath string) {
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if info.IsDir() {
		h.listDirectory(w, fullPath, reqPath)
	} else {
		h.readFile(w, fullPath)
	}
}

// listDirectory returns directory contents as JSON
func (h *FSHandler) listDirectory(w http.ResponseWriter, fullPath, reqPath string) {
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var files []FileInfo
	for _, entry := range entries {
		// Skip hidden files and .devport directory
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		entryPath := filepath.Join(reqPath, entry.Name())
		// Use forward slashes for consistency
		entryPath = filepath.ToSlash(entryPath)

		files = append(files, FileInfo{
			Name:    entry.Name(),
			Path:    entryPath,
			IsDir:   entry.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime().Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	// Sort: directories first, then by name
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return files[i].Name < files[j].Name
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"path":  reqPath,
		"files": files,
	})
}

// readFile returns file contents
func (h *FSHandler) readFile(w http.ResponseWriter, fullPath string) {
	file, err := os.Open(fullPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Detect content type
	ext := filepath.Ext(fullPath)
	contentType := getContentType(ext)
	w.Header().Set("Content-Type", contentType)

	io.Copy(w, file)
}

// handlePut handles PUT requests - write file
func (h *FSHandler) handlePut(w http.ResponseWriter, r *http.Request, fullPath string) {
	// Ensure parent directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write file
	if err := os.WriteFile(fullPath, body, 0644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// handleDelete handles DELETE requests - delete file or directory
func (h *FSHandler) handleDelete(w http.ResponseWriter, r *http.Request, fullPath string) {
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if info.IsDir() {
		err = os.RemoveAll(fullPath)
	} else {
		err = os.Remove(fullPath)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// getContentType returns MIME type based on file extension
func getContentType(ext string) string {
	contentTypes := map[string]string{
		".html": "text/html; charset=utf-8",
		".css":  "text/css; charset=utf-8",
		".js":   "application/javascript; charset=utf-8",
		".ts":   "application/typescript; charset=utf-8",
		".tsx":  "application/typescript; charset=utf-8",
		".jsx":  "application/javascript; charset=utf-8",
		".json": "application/json; charset=utf-8",
		".xml":  "application/xml; charset=utf-8",
		".md":   "text/markdown; charset=utf-8",
		".txt":  "text/plain; charset=utf-8",
		".go":   "text/x-go; charset=utf-8",
		".py":   "text/x-python; charset=utf-8",
		".rs":   "text/x-rust; charset=utf-8",
		".java": "text/x-java; charset=utf-8",
		".c":    "text/x-c; charset=utf-8",
		".cpp":  "text/x-c++; charset=utf-8",
		".h":    "text/x-c; charset=utf-8",
		".hpp":  "text/x-c++; charset=utf-8",
		".svg":  "image/svg+xml",
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
		".webp": "image/webp",
		".ico":  "image/x-icon",
	}

	if ct, ok := contentTypes[strings.ToLower(ext)]; ok {
		return ct
	}
	return "text/plain; charset=utf-8"
}
