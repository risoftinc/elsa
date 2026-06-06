package dbdesign

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
	s := &Server{store: store, mux: http.NewServeMux(), static: sub}
	s.routes()
	return s, nil
}

func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) routes() {
	s.mux.HandleFunc("/", s.handleIndex)
	s.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(s.static))))

	s.mux.HandleFunc("/api/projects", s.handleProjects)
	s.mux.HandleFunc("/api/projects/", s.handleProjectByID)
	s.mux.HandleFunc("/api/tables/", s.handleTableRoutes)
	s.mux.HandleFunc("/api/columns/", s.handleColumnRoutes)
	s.mux.HandleFunc("/api/indexes/", s.handleIndexRoutes)
	s.mux.HandleFunc("/api/relations/", s.handleRelationRoutes)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data, _ := fs.ReadFile(s.static, "index.html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := s.store.ListProjects()
		if err != nil {
			writeError(w, 500, err)
			return
		}
		writeJSON(w, 200, list)
	case http.MethodPost:
		var req struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Dialect     string `json:"dialect"`
		}
		if err := readJSON(r, &req); err != nil {
			writeError(w, 400, err)
			return
		}
		p, err := s.store.CreateProject(req.Name, req.Description, req.Dialect)
		if err != nil {
			writeError(w, 400, err)
			return
		}
		writeJSON(w, 201, p)
	default:
		http.Error(w, "method not allowed", 405)
	}
}

func (s *Server) handleProjectByID(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	rest = strings.TrimSuffix(rest, "/")
	parts := strings.Split(rest, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	projectID := uint(id)

	if len(parts) >= 2 {
		switch parts[1] {
		case "schema":
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", 405)
				return
			}
			schema, err := s.store.GetSchema(projectID)
			if err != nil {
				writeErr(w, err)
				return
			}
			writeJSON(w, 200, schema)
			return
		case "export":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", 405)
				return
			}
			s.handleExport(w, r, projectID)
			return
		case "import":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", 405)
				return
			}
			s.handleImport(w, r, projectID)
			return
		case "tables":
			if r.Method == http.MethodPost {
				var req struct {
					Name string  `json:"name"`
					PosX float64 `json:"pos_x"`
					PosY float64 `json:"pos_y"`
				}
				if err := readJSON(r, &req); err != nil {
					writeError(w, 400, err)
					return
				}
				t, err := s.store.CreateTable(projectID, req.Name, req.PosX, req.PosY)
				if err != nil {
					writeError(w, 400, err)
					return
				}
				writeJSON(w, 201, t)
				return
			}
			http.Error(w, "method not allowed", 405)
			return
		case "relations":
			if r.Method == http.MethodPost {
				var req RelationPayload
				if err := readJSON(r, &req); err != nil {
					writeError(w, 400, err)
					return
				}
				rel, err := s.store.CreateRelation(projectID, req)
				if err != nil {
					writeError(w, 400, err)
					return
				}
				writeJSON(w, 201, rel)
				return
			}
			http.Error(w, "method not allowed", 405)
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		p, err := s.store.GetProject(projectID)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, 200, p)
	case http.MethodPut:
		var req struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Dialect     string `json:"dialect"`
		}
		if err := readJSON(r, &req); err != nil {
			writeError(w, 400, err)
			return
		}
		p, err := s.store.UpdateProject(projectID, req.Name, req.Description, req.Dialect)
		if err != nil {
			writeError(w, 400, err)
			return
		}
		writeJSON(w, 200, p)
	case http.MethodDelete:
		if err := s.store.DeleteProject(projectID); err != nil {
			writeErr(w, err)
			return
		}
		w.WriteHeader(204)
	default:
		http.Error(w, "method not allowed", 405)
	}
}

func (s *Server) handleImport(w http.ResponseWriter, r *http.Request, projectID uint) {
	var req struct {
		SQL     string `json:"sql"`
		Replace bool   `json:"replace"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, 400, err)
		return
	}
	if strings.TrimSpace(req.SQL) == "" {
		writeError(w, 400, fmt.Errorf("sql is required"))
		return
	}
	result, err := s.store.ImportSQL(projectID, req.SQL, req.Replace)
	if err != nil {
		writeError(w, 400, err)
		return
	}
	writeJSON(w, 200, result)
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request, projectID uint) {
	var req struct {
		Dialect string `json:"dialect"`
	}
	_ = readJSONOptional(r, &req)

	schema, err := s.store.GetSchema(projectID)
	if err != nil {
		writeErr(w, err)
		return
	}
	sql, err := ExportSQL(schema, req.Dialect)
	if err != nil {
		writeError(w, 400, err)
		return
	}
	dialect := req.Dialect
	if dialect == "" {
		dialect = schema.Project.Dialect
	}
	writeJSON(w, 200, ExportResult{
		SQL:      sql,
		Dialect:  dialect,
		Filename: ExportFilename(schema.Project.Name, dialect),
	})
}

func (s *Server) handleTableRoutes(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/tables/")
	rest = strings.TrimSuffix(rest, "/")
	parts := strings.Split(rest, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	tableID, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if len(parts) == 2 && parts[1] == "columns" && r.Method == http.MethodPost {
		var p ColumnPayload
		if err := readJSON(r, &p); err != nil {
			writeError(w, 400, err)
			return
		}
		c, err := s.store.CreateColumn(uint(tableID), p)
		if err != nil {
			writeError(w, 400, err)
			return
		}
		writeJSON(w, 201, c)
		return
	}

	if len(parts) == 2 && parts[1] == "indexes" && r.Method == http.MethodPost {
		var p IndexPayload
		if err := readJSON(r, &p); err != nil {
			writeError(w, 400, err)
			return
		}
		idx, err := s.store.CreateIndex(uint(tableID), p)
		if err != nil {
			writeError(w, 400, err)
			return
		}
		writeJSON(w, 201, idx)
		return
	}

	if len(parts) != 1 {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodPut:
		var req struct {
			Name string   `json:"name"`
			PosX *float64 `json:"pos_x"`
			PosY *float64 `json:"pos_y"`
		}
		if err := readJSON(r, &req); err != nil {
			writeError(w, 400, err)
			return
		}
		t, err := s.store.UpdateTable(uint(tableID), req.Name, req.PosX, req.PosY)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, 200, t)
	case http.MethodDelete:
		if err := s.store.DeleteTable(uint(tableID)); err != nil {
			writeErr(w, err)
			return
		}
		w.WriteHeader(204)
	default:
		http.Error(w, "method not allowed", 405)
	}
}

func (s *Server) handleColumnRoutes(w http.ResponseWriter, r *http.Request) {
	id, ok := parseTrailingID(r.URL.Path, "/api/columns/")
	if !ok {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var p ColumnPayload
		if err := readJSON(r, &p); err != nil {
			writeError(w, 400, err)
			return
		}
		c, err := s.store.UpdateColumn(id, p)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, 200, c)
	case http.MethodDelete:
		if err := s.store.DeleteColumn(id); err != nil {
			writeErr(w, err)
			return
		}
		w.WriteHeader(204)
	default:
		http.Error(w, "method not allowed", 405)
	}
}

func (s *Server) handleIndexRoutes(w http.ResponseWriter, r *http.Request) {
	id, ok := parseTrailingID(r.URL.Path, "/api/indexes/")
	if !ok {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var p IndexPayload
		if err := readJSON(r, &p); err != nil {
			writeError(w, 400, err)
			return
		}
		idx, err := s.store.UpdateIndex(id, p)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, 200, idx)
	case http.MethodDelete:
		if err := s.store.DeleteIndex(id); err != nil {
			writeErr(w, err)
			return
		}
		w.WriteHeader(204)
	default:
		http.Error(w, "method not allowed", 405)
	}
}

func (s *Server) handleRelationRoutes(w http.ResponseWriter, r *http.Request) {
	id, ok := parseTrailingID(r.URL.Path, "/api/relations/")
	if !ok {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var req RelationPayload
		if err := readJSON(r, &req); err != nil {
			writeError(w, 400, err)
			return
		}
		rel, err := s.store.UpdateRelation(id, req)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, 200, rel)
	case http.MethodDelete:
		if err := s.store.DeleteRelation(id); err != nil {
			writeErr(w, err)
			return
		}
		w.WriteHeader(204)
	default:
		http.Error(w, "method not allowed", 405)
	}
}

func parseTrailingID(path, prefix string) (uint, bool) {
	rest := strings.TrimPrefix(path, prefix)
	rest = strings.TrimSuffix(rest, "/")
	if rest == "" {
		return 0, false
	}
	n, err := strconv.ParseUint(rest, 10, 64)
	if err != nil {
		return 0, false
	}
	return uint(n), true
}

func writeErr(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) {
		writeError(w, 404, err)
		return
	}
	writeError(w, 500, err)
}

func readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	return dec.Decode(v)
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
