package web

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/spndxyz/quiz/internal/domain"
)

//go:embed templates static
var files embed.FS

// base carries the fields every page needs. Page structs embed it.
type base struct {
	User  *domain.User
	Flash string
}

// renderer holds the parsed templates: one set per page, plus a set that holds
// only the partials, used for the fragments htmx swaps in.
type renderer struct {
	pages     map[string]*template.Template
	fragments *template.Template
}

var funcs = template.FuncMap{
	"inc": func(i int) int { return i + 1 },
}

func newRenderer() (*renderer, error) {
	pageFiles, err := fs.Glob(files, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("glob pages: %w", err)
	}

	r := &renderer{pages: make(map[string]*template.Template, len(pageFiles))}
	for _, page := range pageFiles {
		if page == "templates/base.html" {
			continue
		}
		tmpl, err := template.New("base.html").Funcs(funcs).ParseFS(files,
			"templates/base.html", "templates/partials/*.html", page)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", page, err)
		}
		r.pages[pageName(page)] = tmpl
	}

	r.fragments, err = template.New("fragments").Funcs(funcs).ParseFS(files, "templates/partials/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse partials: %w", err)
	}
	return r, nil
}

// pageName turns "templates/login.html" into "login".
func pageName(path string) string {
	return path[len("templates/") : len(path)-len(".html")]
}

// page renders a full page. It buffers first so a template error cannot leave
// a half-written response behind.
func (r *renderer) page(w http.ResponseWriter, status int, name string, data any) error {
	tmpl, ok := r.pages[name]
	if !ok {
		return fmt.Errorf("unknown page %q", name)
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "base", data); err != nil {
		return fmt.Errorf("render page %s: %w", name, err)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, err := buf.WriteTo(w)
	return err
}

// fragment renders a single partial, for htmx to swap into the page.
func (r *renderer) fragment(w http.ResponseWriter, name string, data any) error {
	var buf bytes.Buffer
	if err := r.fragments.ExecuteTemplate(&buf, name, data); err != nil {
		return fmt.Errorf("render fragment %s: %w", name, err)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err := buf.WriteTo(w)
	return err
}

// staticFS is the sub-tree served under /static/.
func staticFS() (fs.FS, error) {
	sub, err := fs.Sub(files, "static")
	if err != nil {
		return nil, fmt.Errorf("static sub fs: %w", err)
	}
	return sub, nil
}
