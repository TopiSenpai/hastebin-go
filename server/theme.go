package server

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"

	"go.gopad.dev/go-tree-sitter-highlight/html"

	"github.com/topi314/gobin/v2/internal/ezhttp"
)

func init() {
	registerTheme(Theme{
		Name:        "default",
		ColorScheme: "dark",
		Theme:       html.DefaultTheme(),
	})
}

var themes = make(map[string]Theme)

func registerTheme(theme Theme) {
	themes[theme.Name] = theme
}

type Theme struct {
	Name         string
	ColorScheme  string
	Theme        html.Theme
	CaptureNames []string
}

func getTheme(r *http.Request) Theme {
	var themeName string
	if themeCookie, err := r.Cookie("theme"); err == nil {
		themeName = themeCookie.Value
	}
	queryTheme := r.URL.Query().Get("theme")
	if queryTheme != "" {
		themeName = queryTheme
	}

	theme, ok := themes[themeName]
	if !ok {
		return themes["default"]
	}

	return theme
}

func (s *Server) ThemeCSS(w http.ResponseWriter, r *http.Request) {
	theme := getTheme(r)
	cssBuff := s.themeCSS(theme)

	w.Header().Set(ezhttp.HeaderContentType, ezhttp.ContentTypeCSS)
	w.Header().Set(ezhttp.HeaderContentLength, strconv.Itoa(len(cssBuff)))
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write([]byte(cssBuff))
}

func (s *Server) themeCSS(theme Theme) string {
	cssBuff := new(bytes.Buffer)
	_, _ = fmt.Fprintf(cssBuff, "html{color-scheme: %s;}", theme.ColorScheme)
	_, _ = fmt.Fprint(cssBuff, ":root{")
	_, _ = fmt.Fprintf(cssBuff, "--bg-primary: %s;", theme.Theme.Background0)
	_, _ = fmt.Fprintf(cssBuff, "--bg-secondary: %s;", theme.Theme.Background1)
	_, _ = fmt.Fprintf(cssBuff, "--nav-button-bg: %s;", theme.Theme.Background2)
	_, _ = fmt.Fprintf(cssBuff, "--text-primary: %s;", theme.Theme.Text0)
	_, _ = fmt.Fprintf(cssBuff, "--text-secondary: %s;", theme.Theme.Text1)
	// _, _ = fmt.Fprintf(cssBuff, "--bg-scrollbar: %s;", background.Background.BrightenOrDarken(0.1).String())
	// _, _ = fmt.Fprintf(cssBuff, "--bg-scrollbar-thumb: %s;", background.Background.BrightenOrDarken(0.2).String())
	// _, _ = fmt.Fprintf(cssBuff, "--bg-scrollbar-thumb-hover: %s;", background.Background.BrightenOrDarken(0.3).String())
	_, _ = fmt.Fprint(cssBuff, "}")

	_ = s.htmlRenderer.RenderCSS(cssBuff, theme.Theme)
	return cssBuff.String()
}
