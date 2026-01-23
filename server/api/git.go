package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os/exec"
	"strings"
)

// GitHandler handles Git operations
type GitHandler struct {
	workDir   string
	authToken string
}

// NewGitHandler creates a new Git handler
func NewGitHandler(workDir, authToken string) *GitHandler {
	return &GitHandler{
		workDir:   workDir,
		authToken: authToken,
	}
}

// DiffFile represents a file in the diff
type DiffFile struct {
	Path      string `json:"path"`
	Status    string `json:"status"` // "added", "modified", "deleted", "renamed"
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

// DiffResponse represents the git diff response
type DiffResponse struct {
	Branch      string     `json:"branch"`
	Files       []DiffFile `json:"files"`
	Diff        string     `json:"diff"`
	HasChanges  bool       `json:"has_changes"`
	Staged      []DiffFile `json:"staged"`
	StagedDiff  string     `json:"staged_diff"`
}

// StatusResponse represents git status
type StatusResponse struct {
	Branch        string   `json:"branch"`
	IsRepo        bool     `json:"is_repo"`
	HasChanges    bool     `json:"has_changes"`
	Staged        []string `json:"staged"`
	Unstaged      []string `json:"unstaged"`
	Untracked     []string `json:"untracked"`
}

// ServeHTTP implements http.Handler
func (h *GitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	token := r.Header.Get("Authorization")
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	if strings.TrimPrefix(token, "Bearer ") != h.authToken {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Route based on path
	path := strings.TrimPrefix(r.URL.Path, "/api/git")

	switch {
	case path == "/status" && r.Method == http.MethodGet:
		h.handleStatus(w, r)
	case path == "/diff" && r.Method == http.MethodGet:
		h.handleDiff(w, r)
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

// handleStatus returns git status
func (h *GitHandler) handleStatus(w http.ResponseWriter, r *http.Request) {
	response := StatusResponse{
		IsRepo: h.isGitRepo(),
	}

	if !response.IsRepo {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get current branch
	branch, _ := h.runGit("rev-parse", "--abbrev-ref", "HEAD")
	response.Branch = strings.TrimSpace(branch)

	// Get status
	status, _ := h.runGit("status", "--porcelain")
	lines := strings.Split(strings.TrimSpace(status), "\n")

	for _, line := range lines {
		if len(line) < 3 {
			continue
		}

		indexStatus := line[0]
		workStatus := line[1]
		file := strings.TrimSpace(line[3:])

		// Staged changes
		if indexStatus != ' ' && indexStatus != '?' {
			response.Staged = append(response.Staged, file)
		}

		// Unstaged changes
		if workStatus != ' ' && workStatus != '?' {
			response.Unstaged = append(response.Unstaged, file)
		}

		// Untracked files
		if indexStatus == '?' && workStatus == '?' {
			response.Untracked = append(response.Untracked, file)
		}
	}

	response.HasChanges = len(response.Staged) > 0 || len(response.Unstaged) > 0 || len(response.Untracked) > 0

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleDiff returns git diff
func (h *GitHandler) handleDiff(w http.ResponseWriter, r *http.Request) {
	response := DiffResponse{}

	if !h.isGitRepo() {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get current branch
	branch, _ := h.runGit("rev-parse", "--abbrev-ref", "HEAD")
	response.Branch = strings.TrimSpace(branch)

	// Get unstaged diff
	diff, _ := h.runGit("diff")
	response.Diff = diff
	response.Files = h.parseDiffStat(false)

	// Get staged diff
	stagedDiff, _ := h.runGit("diff", "--cached")
	response.StagedDiff = stagedDiff
	response.Staged = h.parseDiffStat(true)

	response.HasChanges = len(response.Files) > 0 || len(response.Staged) > 0

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// parseDiffStat parses diff --stat output
func (h *GitHandler) parseDiffStat(staged bool) []DiffFile {
	var args []string
	if staged {
		args = []string{"diff", "--cached", "--numstat"}
	} else {
		args = []string{"diff", "--numstat"}
	}

	output, err := h.runGit(args...)
	if err != nil {
		return nil
	}

	var files []DiffFile
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		var additions, deletions int
		if parts[0] != "-" {
			var n int
			for _, ch := range parts[0] {
				if ch >= '0' && ch <= '9' {
					n = n*10 + int(ch-'0')
				}
			}
			additions = n
		}
		if parts[1] != "-" {
			var n int
			for _, ch := range parts[1] {
				if ch >= '0' && ch <= '9' {
					n = n*10 + int(ch-'0')
				}
			}
			deletions = n
		}

		path := parts[2]
		status := "modified"
		if additions > 0 && deletions == 0 {
			status = "added"
		} else if additions == 0 && deletions > 0 {
			status = "deleted"
		}

		files = append(files, DiffFile{
			Path:      path,
			Status:    status,
			Additions: additions,
			Deletions: deletions,
		})
	}

	return files
}

// isGitRepo checks if the work directory is a git repository
func (h *GitHandler) isGitRepo() bool {
	_, err := h.runGit("rev-parse", "--git-dir")
	return err == nil
}

// runGit runs a git command and returns the output
func (h *GitHandler) runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = h.workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return stderr.String(), err
	}

	return stdout.String(), nil
}
