package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"minigit/internal/repository"
)

func TestServerEndpoints(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "minigit-server-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repo, err := repository.InitRepository(tempDir)
	if err != nil {
		t.Fatalf("InitRepository failed: %v", err)
	}

	// Add file and commit
	filePath := filepath.Join(tempDir, "hello.txt")
	os.WriteFile(filePath, []byte("Hello Server\n"), 0644)
	repo.Add([]string{"."})
	commitRes, err := repo.Commit("Initial web server commit", "ServerDev", "dev@server.org")
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	srv := NewServer(tempDir)
	mux := http.NewServeMux()
	srv.RegisterHandlers(mux)

	// 1. Test GET /
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected GET / to return 200, got %d", w.Code)
	}

	// 2. Test GET /api/projects
	req = httptest.NewRequest("GET", "/api/projects", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected GET /api/projects to return 200, got %d", w.Code)
	}

	// 3. Test GET /api/dashboard
	req = httptest.NewRequest("GET", "/api/dashboard", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected GET /api/dashboard to return 200, got %d", w.Code)
	}

	var dashMap map[string]interface{}
	json.NewDecoder(w.Body).Decode(&dashMap)
	if dashMap["active_branch"] != "main" {
		t.Errorf("expected active_branch main, got %v", dashMap["active_branch"])
	}

	// 4. Test GET /api/history
	req = httptest.NewRequest("GET", "/api/history", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected GET /api/history to return 200, got %d", w.Code)
	}

	// 5. Test GET /api/branches
	req = httptest.NewRequest("GET", "/api/branches", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected GET /api/branches to return 200, got %d", w.Code)
	}

	// 6. Test GET /api/commit?hash=...
	req = httptest.NewRequest("GET", "/api/commit?hash="+commitRes.Hash, nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected GET /api/commit to return 200, got %d", w.Code)
	}

	// 7. Test GET /api/status
	req = httptest.NewRequest("GET", "/api/status", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected GET /api/status to return 200, got %d", w.Code)
	}
}
