package renderer

import (
	"bytes"
	"html"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dadanrm/hypergon"
)

type RendererHook func(http.ResponseWriter, *http.Request, map[string]any)

type HTMLRenderConfig struct {
	// Layout name declaration.
	// Default: "layout.html"
	Layout string
	// TemplateDirs where you declare where are your html templates are in.
	//
	// Default:  []string{"templates", "templates/partials"}
	TemplateDirs []string
}

// HTMLRender accepts templates and a layout that will parse all necessary html files.
type HTMLRender struct {
	templates *template.Template
	layout    string
	hooks     []RendererHook
}

// The constructor for the HTMLRender.
func NewHTMLRenderer(cfg ...HTMLRenderConfig) *HTMLRender {
	config := HTMLRenderConfig{
		Layout:       "layout.html",
		TemplateDirs: []string{"templates"},
	}

	if len(cfg) > 0 {
		config = cfg[0]
	}

	funcs := template.FuncMap{
		"formatDate": formatDate,
		// This temlate functions handles pointers in templates.
		"deref": func(value any) any {
			if value == nil {
				return ""
			}

			switch v := value.(type) {
			case *string:
				if v == nil {
					return ""
				}
				return *v
			case *int:
				if v == nil {
					return 0
				}
				return *v
			case *bool:
				if v == nil {
					return false
				}
				return *v
			default:
				return value
			}
		},
		"unescape": html.UnescapeString, // WARN: use this with caution
	}

	// Start with an empty template set
	tmpls := template.New("").Funcs(funcs)

	// Recursively parse all .html files in each provided template directory
	for _, tmplDir := range config.TemplateDirs {
		err := filepath.Walk(tmplDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && filepath.Ext(path) == ".html" {
				_, err = tmpls.ParseFiles(path)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			log.Fatalf("Failed to parse templates in %s: %v", tmplDir, err)
		}
	}

	// WARN: Print parsed templates to ensure they are loaded
	//	log.Println("Templates parsed:", tmpls.DefinedTemplates())

	return &HTMLRender{
		templates: tmpls,
		layout:    config.Layout,
		hooks:     []RendererHook{},
	}
}

// Allows adding a hook to modify data globally.
func (r *HTMLRender) AddHook(hook RendererHook) {
	r.hooks = append([]RendererHook{hook}, r.hooks...)
}

// Render renders a html page with layout.
// The name must unique so it wont be a conflict to other html pages.
func (r *HTMLRender) Render(w http.ResponseWriter, req *http.Request, name string, data map[string]any) hypergon.HypergonError {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if data == nil {
		data = make(map[string]any)
	}

	for _, hook := range r.hooks {
		hook(w, req, data)
	}

	// Render the content template to a buffer
	var contentBuffer bytes.Buffer
	err := r.templates.ExecuteTemplate(&contentBuffer, name, data)
	if err != nil {
		return hypergon.HttpError(http.StatusInternalServerError, "Content render error: "+err.Error())
	}

	safeContent := sanitizeHTML(contentBuffer.String())

	err = r.templates.ExecuteTemplate(w, r.layout, map[string]any{
		"Content": template.HTML(safeContent),
		"Data":    data,
	})
	if err != nil {
		return hypergon.HttpError(http.StatusInternalServerError, "Content render error: "+err.Error())
	}

	return nil
}

// RenderPartial renders a partial template without the layout
// The name must unique so it wont be a conflict to other partial htmls.
func (r *HTMLRender) RenderPartial(w http.ResponseWriter, name string, data any) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Render just the partial template directly to the response
	return r.templates.ExecuteTemplate(w, name, data)
}

func sanitizeHTML(input string) string {
	// Regex to match all <script> tags (both inline and external)
	re := regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)

	// Replace matches using a custom function
	safe := re.ReplaceAllStringFunc(input, func(tag string) string {
		// Check if the <script> tag contains 'src=' (indicating an external script)
		if strings.Contains(strings.ToLower(tag), " src=") {
			return tag // Keep external script tags
		}
		return "" // Remove inline scripts
	})

	// Escape remaining dangerous characters
	return safe
}

func formatDate(date *time.Time, layout string) string {
	if date == nil {
		return "Invalid Date"
	}

	return date.Format(layout)
}
