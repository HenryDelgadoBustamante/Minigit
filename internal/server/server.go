package server

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"minigit/internal/object"
	"minigit/internal/repository"
)

type Server struct {
	Projects       map[string]*repository.Repository
	DefaultProject string
	StartDir       string
}

func NewServer(startDir string) *Server {
	projects := DiscoverRepositories(startDir)
	defaultProj := ""
	for name := range projects {
		defaultProj = name
		break
	}
	return &Server{
		Projects:       projects,
		DefaultProject: defaultProj,
		StartDir:       startDir,
	}
}

// DiscoverRepositories scans startDir (and subdirectories up to 3 levels deep) for .minigit repos.
func DiscoverRepositories(startDir string) map[string]*repository.Repository {
	projects := make(map[string]*repository.Repository)

	absStart, err := filepath.Abs(startDir)
	if err != nil {
		absStart = startDir
	}

	// 1. Check if startDir itself is a MiniGit repo
	if _, err := os.Stat(filepath.Join(absStart, ".minigit")); err == nil {
		name := filepath.Base(absStart)
		projects[name] = repository.OpenRepository(absStart)
	}

	// 2. Scan subdirectories (including proyectos_minigit/)
	filepath.Walk(absStart, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			// Skip hidden folders except root check
			if strings.HasPrefix(info.Name(), ".") && path != absStart {
				return filepath.SkipDir
			}

			// Check depth relative to startDir
			rel, _ := filepath.Rel(absStart, path)
			depth := len(filepath.SplitList(rel))
			if depth > 4 {
				return filepath.SkipDir
			}

			if _, err := os.Stat(filepath.Join(path, ".minigit")); err == nil {
				name := info.Name()
				if path == absStart {
					name = filepath.Base(absStart)
				}
				if _, exists := projects[name]; !exists {
					projects[name] = repository.OpenRepository(path)
				}
				return filepath.SkipDir // Don't recurse into .minigit or sub-repos
			}
		}
		return nil
	})

	return projects
}

// getRepo returns the repository requested in query param ?repo=..., or the default repository.
func (s *Server) getRepo(r *http.Request) *repository.Repository {
	repoName := r.URL.Query().Get("repo")
	if repoName != "" {
		if repo, exists := s.Projects[repoName]; exists {
			return repo
		}
	}
	if s.DefaultProject != "" {
		return s.Projects[s.DefaultProject]
	}
	return nil
}

// StartWebServer launches the HTTP server and opens the local browser.
func (s *Server) StartWebServer(port int) error {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("error al iniciar servidor en puerto %d: %w", port, err)
	}

	url := fmt.Sprintf("http://localhost:%d", port)
	fmt.Printf("\n🚀 Servidor web local de MiniGit iniciado correctamente.\n")
	fmt.Printf("🌐 Abre en tu navegador: %s\n", url)
	fmt.Printf("📁 Repositorios descubiertos (%d): ", len(s.Projects))

	var names []string
	for name := range s.Projects {
		names = append(names, name)
	}
	fmt.Printf("%s\n", strings.Join(names, ", "))
	fmt.Printf("📌 Presiona Ctrl+C para detener el servidor web.\n\n")

	openBrowser(url)

	mux := http.NewServeMux()
	s.RegisterHandlers(mux)

	return http.Serve(listener, mux)
}

func (s *Server) RegisterHandlers(mux *http.ServeMux) {
	// Static files from embedded FS
	staticFS, err := fs.Sub(StaticAssets, "static")
	if err == nil {
		fileServer := http.FileServer(http.FS(staticFS))
		mux.Handle("/static/", http.StripPrefix("/static/", fileServer))
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data, err := StaticAssets.ReadFile("static/index.html")
		if err != nil {
			http.Error(w, "Error cargando interfaz web", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})

	// API JSON Endpoints
	mux.HandleFunc("/api/projects", s.handleProjects)
	mux.HandleFunc("/api/dashboard", s.handleDashboard)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/history", s.handleHistory)
	mux.HandleFunc("/api/branches", s.handleBranches)
	mux.HandleFunc("/api/commit", s.handleCommit)
	mux.HandleFunc("/api/tree", s.handleTree)
	mux.HandleFunc("/api/file", s.handleFile)
	mux.HandleFunc("/api/diff", s.handleDiff)
}

func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	type projItem struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	var list []projItem
	for name, repo := range s.Projects {
		list = append(list, projItem{Name: name, Path: repo.Root})
	}
	writeJSON(w, map[string]interface{}{
		"active":   s.DefaultProject,
		"projects": list,
	})
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	repo := s.getRepo(r)
	if repo == nil {
		http.Error(w, "No hay repositorios disponibles", http.StatusNotFound)
		return
	}

	head, _ := repository.ReadHEAD(repo.Root)
	activeBranch := "detached"
	if head != nil && head.Type == repository.HEADTypeBranch {
		activeBranch = head.Branch
	}

	history, _ := repo.GetCommitHistory()
	branches, _ := repo.ListBranches()
	status, _ := repo.GetStatus()

	stagedCount := 0
	unstagedCount := 0
	untrackedCount := 0
	if status != nil {
		stagedCount = len(status.Staged)
		unstagedCount = len(status.Unstaged)
		untrackedCount = len(status.Untracked)
	}

	var latestCommit map[string]interface{}
	if len(history) > 0 {
		latestCommit = map[string]interface{}{
			"hash":      history[0].Hash,
			"author":    history[0].AuthorName,
			"timestamp": history[0].Timestamp,
			"message":   history[0].Message,
		}
	}

	response := map[string]interface{}{
		"repo_name":       filepath.Base(repo.Root),
		"repo_root":       repo.Root,
		"active_branch":   activeBranch,
		"total_commits":   len(history),
		"total_branches":  len(branches),
		"staged_count":    stagedCount,
		"unstaged_count":  unstagedCount,
		"untracked_count": untrackedCount,
		"latest_commit":   latestCommit,
	}

	writeJSON(w, response)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	repo := s.getRepo(r)
	if repo == nil {
		http.Error(w, "Repositorio no encontrado", http.StatusNotFound)
		return
	}
	status, err := repo.GetStatus()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, status)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	repo := s.getRepo(r)
	if repo == nil {
		writeJSON(w, []interface{}{})
		return
	}
	history, err := repo.GetCommitHistory()
	if err != nil {
		writeJSON(w, []interface{}{})
		return
	}
	writeJSON(w, history)
}

func (s *Server) handleBranches(w http.ResponseWriter, r *http.Request) {
	repo := s.getRepo(r)
	if repo == nil {
		writeJSON(w, map[string]interface{}{"active": "", "branches": []interface{}{}})
		return
	}
	branches, _ := repo.ListBranches()
	head, _ := repository.ReadHEAD(repo.Root)
	active := ""
	if head != nil && head.Type == repository.HEADTypeBranch {
		active = head.Branch
	}

	type branchInfo struct {
		Name   string `json:"name"`
		Commit string `json:"commit"`
	}

	var list []branchInfo
	for _, b := range branches {
		commitHash, _ := repository.ReadBranchCommit(repo.Root, b)
		list = append(list, branchInfo{Name: b, Commit: commitHash})
	}

	writeJSON(w, map[string]interface{}{
		"active":   active,
		"branches": list,
	})
}

func (s *Server) handleCommit(w http.ResponseWriter, r *http.Request) {
	repo := s.getRepo(r)
	if repo == nil {
		http.Error(w, "Repositorio no encontrado", http.StatusNotFound)
		return
	}

	hash := r.URL.Query().Get("hash")
	if hash == "" {
		hash = "HEAD"
	}

	res, err := repo.InspectObject(hash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if res.Type != object.TypeCommit {
		http.Error(w, "El objeto no es un commit", http.StatusBadRequest)
		return
	}

	commit := res.Commit
	response := map[string]interface{}{
		"hash":         res.FullHash,
		"tree":         commit.Tree,
		"parent":       commit.Parent,
		"author_name":  commit.AuthorName,
		"author_email": commit.AuthorMail,
		"timestamp":    commit.CreatedAt,
		"message":      commit.Message,
		"diff":         res.CommitDiff,
	}

	writeJSON(w, response)
}

func (s *Server) handleTree(w http.ResponseWriter, r *http.Request) {
	repo := s.getRepo(r)
	if repo == nil {
		writeJSON(w, []interface{}{})
		return
	}

	hash := r.URL.Query().Get("hash")
	if hash == "" {
		headHash, err := repo.GetHeadCommitHash()
		if err != nil || headHash == "" {
			writeJSON(w, []interface{}{})
			return
		}
		commit, _, err := repo.GetCommitByHash(headHash)
		if err != nil {
			writeJSON(w, []interface{}{})
			return
		}
		hash = commit.Tree
	}

	raw, _, err := repo.Objects.ReadObject(hash)
	if err != nil {
		http.Error(w, "Tree no encontrado", http.StatusNotFound)
		return
	}

	tree, err := object.DecodeTree(raw)
	if err != nil {
		http.Error(w, "Formato de tree inválido", http.StatusInternalServerError)
		return
	}

	type treeItem struct {
		Name string `json:"name"`
		Hash string `json:"hash"`
		Type string `json:"type"`
		Mode string `json:"mode"`
	}

	var items []treeItem
	for _, entry := range tree.Entries {
		items = append(items, treeItem{
			Name: entry.Name,
			Hash: entry.Hash,
			Type: entry.Type,
			Mode: fmt.Sprintf("%06o", entry.Mode),
		})
	}

	writeJSON(w, items)
}

func (s *Server) handleFile(w http.ResponseWriter, r *http.Request) {
	repo := s.getRepo(r)
	if repo == nil {
		http.Error(w, "Repositorio no encontrado", http.StatusNotFound)
		return
	}

	hash := r.URL.Query().Get("hash")
	if hash == "" {
		http.Error(w, "Se requiere el hash del archivo", http.StatusBadRequest)
		return
	}

	content, err := repo.GetBlobContent(hash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	writeJSON(w, map[string]interface{}{
		"hash":    hash,
		"content": string(content),
	})
}

func (s *Server) handleDiff(w http.ResponseWriter, r *http.Request) {
	repo := s.getRepo(r)
	if repo == nil {
		http.Error(w, "Repositorio no encontrado", http.StatusNotFound)
		return
	}

	c1 := r.URL.Query().Get("commit1")
	c2 := r.URL.Query().Get("commit2")

	if c1 == "" || c2 == "" {
		http.Error(w, "Se requieren los parámetros commit1 y commit2", http.StatusBadRequest)
		return
	}

	diffRes, err := repo.CompareCommits(c1, c2)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	type fileDiffItem struct {
		Type      string   `json:"Type"`
		Path      string   `json:"Path"`
		OldPath   string   `json:"OldPath,omitempty"`
		DiffLines []string `json:"diff_lines,omitempty"`
	}

	var items []fileDiffItem
	for _, change := range diffRes.Changes {
		oldData, _ := repo.GetBlobContent(change.OldHash)
		newData, _ := repo.GetBlobContent(change.NewHash)
		lines, _ := repository.GetFileDiffLines(oldData, newData)

		items = append(items, fileDiffItem{
			Type:      string(change.Type),
			Path:      change.Path,
			OldPath:   change.OldPath,
			DiffLines: lines,
		})
	}

	writeJSON(w, map[string]interface{}{
		"commit1": c1,
		"commit2": c2,
		"changes": items,
	})
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(data)
}

func openBrowser(url string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	exec.Command(cmd, args...).Start()
}
