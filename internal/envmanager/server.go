package envmanager

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
)

// Server serves the env manager API and embedded web UI
type Server struct {
	store  *Store
	mux    *http.ServeMux
	static fs.FS
}

func NewServer(store *Store) (*Server, error) {
	sub, err := fs.Sub(WebFS, "web")
	if err != nil {
		return nil, err
	}

	s := &Server{
		store:  store,
		mux:    http.NewServeMux(),
		static: sub,
	}
	s.routes()
	return s, nil
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	s.mux.HandleFunc("/", s.handleIndex)
	s.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(s.static))))

	s.mux.HandleFunc("/api/environments", s.handleEnvironments)
	s.mux.HandleFunc("/api/environments/", s.handleEnvironmentByID)
	s.mux.HandleFunc("/api/variables", s.handleVariables)
	s.mux.HandleFunc("/api/variables/", s.handleVariableByID)
	s.mux.HandleFunc("/api/templates", s.handleTemplates)
	s.mux.HandleFunc("/api/templates/", s.handleTemplateByID)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data, err := fs.ReadFile(s.static, "index.html")
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

// --- environments ---

func (s *Server) handleEnvironments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		envs, err := s.store.ListEnvironments()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, envs)
	case http.MethodPost:
		var req struct {
			Name      string `json:"name"`
			SortOrder int    `json:"sort_order"`
		}
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		env, err := s.store.CreateEnvironment(req.Name, req.SortOrder)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, env)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleEnvironmentByID(w http.ResponseWriter, r *http.Request) {
	id, action, ok := parseSubPath(r.URL.Path, "/api/environments/")
	if !ok {
		http.NotFound(w, r)
		return
	}

	if action == "export" {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.handleExportEnv(w, r, id)
		return
	}

	switch r.Method {
	case http.MethodPut:
		var req struct {
			Name      string `json:"name"`
			SortOrder int    `json:"sort_order"`
		}
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		env, err := s.store.UpdateEnvironment(id, req.Name, req.SortOrder)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, ErrNotFound) {
				status = http.StatusNotFound
			}
			writeError(w, status, err)
			return
		}
		writeJSON(w, http.StatusOK, env)
	case http.MethodDelete:
		if err := s.store.DeleteEnvironment(id); err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, ErrNotFound) {
				status = http.StatusNotFound
			}
			writeError(w, status, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleExportEnv(w http.ResponseWriter, r *http.Request, envID uint) {
	content, err := s.store.ExportDotEnv(envID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}

	envs, _ := s.store.ListEnvironments()
	filename := ".env"
	for _, e := range envs {
		if e.ID == envID {
			filename = DefaultFilename(e.Name)
			break
		}
	}

	var req struct {
		Filename string `json:"filename"`
	}
	_ = readJSONOptional(r, &req)
	if strings.TrimSpace(req.Filename) != "" {
		filename = req.Filename
	}

	writeJSON(w, http.StatusOK, RenderResponse{Content: content, Filename: filename})
}

// --- variables ---

func (s *Server) handleVariables(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		vars, err := s.store.ListVariables()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, vars)
	case http.MethodPost:
		var payload VariablePayload
		if err := readJSON(r, &payload); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		v, err := s.store.CreateVariable(payload)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, v)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleVariableByID(w http.ResponseWriter, r *http.Request) {
	id, _, ok := parseSubPath(r.URL.Path, "/api/variables/")
	if !ok {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		v, err := s.store.GetVariable(id)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, ErrNotFound) {
				status = http.StatusNotFound
			}
			writeError(w, status, err)
			return
		}
		writeJSON(w, http.StatusOK, v)
	case http.MethodPut:
		var payload VariablePayload
		if err := readJSON(r, &payload); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		v, err := s.store.UpdateVariable(id, payload)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, ErrNotFound) {
				status = http.StatusNotFound
			}
			writeError(w, status, err)
			return
		}
		writeJSON(w, http.StatusOK, v)
	case http.MethodDelete:
		if err := s.store.DeleteVariable(id); err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, ErrNotFound) {
				status = http.StatusNotFound
			}
			writeError(w, status, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// --- templates ---

func (s *Server) handleTemplates(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := s.store.ListTemplates()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, list)
	case http.MethodPost:
		var req struct {
			Name string `json:"name"`
			Body string `json:"body"`
		}
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		t, err := s.store.CreateTemplate(req.Name, req.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, t)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleTemplateByID(w http.ResponseWriter, r *http.Request) {
	id, action, ok := parseSubPath(r.URL.Path, "/api/templates/")
	if !ok {
		http.NotFound(w, r)
		return
	}

	if action == "render" {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.handleRenderTemplate(w, r, id)
		return
	}

	switch r.Method {
	case http.MethodGet:
		t, err := s.store.GetTemplate(id)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, ErrNotFound) {
				status = http.StatusNotFound
			}
			writeError(w, status, err)
			return
		}
		writeJSON(w, http.StatusOK, t)
	case http.MethodPut:
		var req struct {
			Name string `json:"name"`
			Body string `json:"body"`
		}
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		t, err := s.store.UpdateTemplate(id, req.Name, req.Body)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, ErrNotFound) {
				status = http.StatusNotFound
			}
			writeError(w, status, err)
			return
		}
		writeJSON(w, http.StatusOK, t)
	case http.MethodDelete:
		if err := s.store.DeleteTemplate(id); err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, ErrNotFound) {
				status = http.StatusNotFound
			}
			writeError(w, status, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleRenderTemplate(w http.ResponseWriter, r *http.Request, templateID uint) {
	var req RenderRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	t, err := s.store.GetTemplate(templateID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}

	data, err := s.store.EnvMap(req.EnvironmentID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}

	content, err := RenderTemplate(t.Body, data)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	filename := strings.TrimSpace(req.Filename)
	if filename == "" {
		envs, _ := s.store.ListEnvironments()
		for _, e := range envs {
			if e.ID == req.EnvironmentID {
				filename = DefaultFilename(e.Name)
				break
			}
		}
		if filename == "" {
			filename = ".env"
		}
	}

	writeJSON(w, http.StatusOK, RenderResponse{Content: content, Filename: filename})
}

func parseSubPath(path, prefix string) (id uint, action string, ok bool) {
	rest := strings.TrimPrefix(path, prefix)
	rest = strings.TrimSuffix(rest, "/")
	if rest == "" {
		return 0, "", false
	}
	parts := strings.Split(rest, "/")
	n, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, "", false
	}
	if len(parts) > 1 {
		return uint(n), parts[1], true
	}
	return uint(n), "", true
}

func readJSON(r *http.Request, v any) error {
	if r.Body == nil {
		return fmt.Errorf("empty body")
	}
	defer r.Body.Close()
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	return nil
}

func readJSONOptional(r *http.Request, v any) error {
	if r.Body == nil {
		return nil
	}
	defer r.Body.Close()
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
