package server

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"go.gopad.dev/go-tree-sitter-highlight/html"
	"gopkg.in/yaml.v3"

	"github.com/topi314/gobin/v2/internal/ezhttp"
)

//func init() {
//	registerTheme(NewTheme("default", "dark", UITheme{
//		StatusBarColor:                 "#F8F8F2",
//		StatusBarBackgroundColor:       "#2B2B2B",
//		StatusBarActiveBackgroundColor: "#3C3C3C",
//	}, html.DefaultTheme()))
//}

var themes = make(map[string]Theme)

func registerTheme(theme Theme) {
	themes[theme.Name] = theme
}

func NewTheme(name string, colorScheme string, uiTheme UITheme, codeTheme html.Theme) Theme {
	names := make([]string, 0, len(codeTheme.CodeStyles))
	for styleName := range codeTheme.CodeStyles {
		names = append(names, styleName)
	}

	return Theme{
		Name:         name,
		ColorScheme:  colorScheme,
		UITheme:      uiTheme,
		CodeTheme:    codeTheme,
		CaptureNames: names,
	}
}

type Theme struct {
	Name         string
	ColorScheme  string
	UITheme      UITheme
	CodeTheme    html.Theme
	CaptureNames []string
}

type UITheme struct {
	StatusBarColor                 string
	StatusBarBackgroundColor       string
	StatusBarActiveBackgroundColor string
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
	_, _ = fmt.Fprintf(cssBuff, "html{\ncolor-scheme: %s;\n}\n", theme.ColorScheme)
	_, _ = fmt.Fprint(cssBuff, ":root{\n")
	_, _ = fmt.Fprintf(cssBuff, "--status-bar-color: %s;\n", theme.UITheme.StatusBarColor)
	_, _ = fmt.Fprintf(cssBuff, "--status-bar-background-color: %s;\n", theme.UITheme.StatusBarBackgroundColor)
	_, _ = fmt.Fprintf(cssBuff, "--status-bar-active-background-color: %s;\n", theme.UITheme.StatusBarActiveBackgroundColor)
	_, _ = fmt.Fprintf(cssBuff, "--code-color: %s;\n", theme.CodeTheme.CodeColor)
	_, _ = fmt.Fprintf(cssBuff, "--code-background-color: %s;\n", theme.CodeTheme.CodeBackgroundColor)
	_, _ = fmt.Fprint(cssBuff, "}\n")

	_ = s.htmlRenderer.RenderCSS(cssBuff, theme.CodeTheme)
	return cssBuff.String()
}

func LoadThemes(themes embed.FS) error {
	return fs.WalkDir(themes, "themes", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		data, err := fs.ReadFile(themes, path)
		if err != nil {
			slog.Error("Error while reading theme file", slog.Any("err", err))
			return nil
		}

		var theme Theme
		if strings.HasPrefix(path, "themes/base16/") {
			if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
				return nil
			}
			theme, err = parseBase16Theme(data)
		} else {
			if !strings.HasSuffix(path, ".toml") {
				return nil
			}
			theme, err = parseTheme(data)
		}

		if err != nil {
			slog.Error("Error while parsing theme", slog.Any("err", err))
			return nil
		}

		registerTheme(theme)
		return nil
	})
}

type tomlTheme struct {
	Name        string `toml:"name"`
	ColorScheme string `toml:"color_scheme"`
	TabSize     int    `toml:"tab_size"`

	Colors map[string]string   `toml:"colors"`
	UI     tomlUITheme         `toml:"ui"`
	Styles map[string]rawStyle `toml:"styles"`
}

type tomlUITheme struct {
	StatusBar                 string `toml:"status_bar"`
	StatusBarBackground       string `toml:"status_bar_background"`
	StatusBarActiveBackground string `toml:"status_bar_active_background"`

	Code           string `toml:"code"`
	CodeBackground string `toml:"code_background"`

	LineNumber           string `toml:"line_number"`
	LineNumberBackground string `toml:"line_number_background"`
	Highlight            string `toml:"highlight"`

	Symbols                 string `toml:"symbols"`
	SymbolsBackground       string `toml:"symbols_background"`
	SymbolsActiveBackground string `toml:"symbols_active_background"`
	SymbolKindBackground    string `toml:"symbol_kind_background"`
}

func parseTheme(data []byte) (Theme, error) {
	var theme tomlTheme
	if err := toml.Unmarshal(data, &theme); err != nil {
		return Theme{}, err
	}

	styles := make(map[string]string, len(theme.Styles))
	for name, style := range theme.Styles {
		styles[name] = style.toCSS(theme.Colors)
	}

	return NewTheme(theme.Name, theme.ColorScheme, UITheme{
		StatusBarColor:                 getColor(theme.Colors, theme.UI.StatusBar),
		StatusBarBackgroundColor:       getColor(theme.Colors, theme.UI.StatusBarBackground),
		StatusBarActiveBackgroundColor: getColor(theme.Colors, theme.UI.StatusBarActiveBackground),
	}, html.Theme{
		TabSize:                      theme.TabSize,
		CodeColor:                    getColor(theme.Colors, theme.UI.Code),
		CodeBackgroundColor:          getColor(theme.Colors, theme.UI.CodeBackground),
		LineNumberColor:              getColor(theme.Colors, theme.UI.LineNumber),
		LineNumberBackgroundColor:    getColor(theme.Colors, theme.UI.LineNumberBackground),
		HighlightColor:               getColor(theme.Colors, theme.UI.Highlight),
		SymbolsColor:                 getColor(theme.Colors, theme.UI.Symbols),
		SymbolsBackgroundColor:       getColor(theme.Colors, theme.UI.SymbolsBackground),
		SymbolsActiveBackgroundColor: getColor(theme.Colors, theme.UI.SymbolsActiveBackground),
		SymbolKindBackgroundColor:    getColor(theme.Colors, theme.UI.SymbolKindBackground),
		CodeStyles:                   styles,
	}), nil
}

type base16Theme struct {
	// Scheme is the name of the scheme.
	Scheme string `yaml:"scheme"`
	// Author is the name of the author.
	Author string `yaml:"author"`
	// Theme is either "light" or "dark".
	ColorScheme string `yaml:"color_scheme"`

	// Base00 Default Background
	Base00 string `yaml:"base00"`
	// Base01 Lighter Background (Used for status bars, line number and folding marks)
	Base01 string `yaml:"base01"`
	// Base02 Selection Background
	Base02 string `yaml:"base02"`
	// Base03 Comments, Invisibles, Line Highlighting
	Base03 string `yaml:"base03"`
	// Base04 Dark Foreground (Used for status bars)
	Base04 string `yaml:"base04"`
	// Base05 Default Foreground, Caret, Delimiters, Operators
	Base05 string `yaml:"base05"`
	// Base06 Light Foreground (Not often used)
	Base06 string `yaml:"base06"`
	// Base07 Light Background (Not often used)
	Base07 string `yaml:"base07"`
	// Base08 Variables, XML Tags, Markup Link Text, Markup Lists, Diff Deleted
	Base08 string `yaml:"base08"`
	// Base09 Integers, Boolean, Constants, XML Attributes, Markup Link Url
	Base09 string `yaml:"base09"`
	// Base0A Classes, Markup Bold, Search Text Background
	Base0A string `yaml:"base0A"`
	// Base0B Strings, Inherited Class, Markup Code, Diff Inserted
	Base0B string `yaml:"base0B"`
	// Base0C Support, Regular Expressions, Escape Characters, Markup Quotes
	Base0C string `yaml:"base0C"`
	// Base0D Functions, Methods, Attribute IDs, Headings
	Base0D string `yaml:"base0D"`
	// Base0E Keywords, Storage, Selector, Markup Italic, Diff Changed
	Base0E string `yaml:"base0E"`
	// Base0F Deprecated, Opening/Closing Embedded Language Tags, e.g. `<?php ?>`
	Base0F string `yaml:"base0F"`
}

func parseBase16Theme(data []byte) (Theme, error) {
	var theme base16Theme
	if err := yaml.Unmarshal(data, &theme); err != nil {
		return Theme{}, err
	}

	styles := make(map[string]string)
	styles["comment"] = color(theme.Base03)

	styles["punctuation.delimiter"] = color(theme.Base05)
	styles["operator"] = color(theme.Base05)

	styles["variable"] = color(theme.Base08)
	styles["markup.link.text"] = color(theme.Base08)
	styles["markup.list"] = color(theme.Base08)
	styles["diff.minus"] = color(theme.Base08)

	styles["constant.numeric.integer"] = color(theme.Base09)
	styles["constant.builtin.boolean"] = color(theme.Base09)
	styles["constant"] = color(theme.Base09)
	styles["markup.link.url"] = color(theme.Base09)

	styles["keyword.storage.type"] = color(theme.Base0A)
	styles["markup.bold"] = color(theme.Base0A) + "font-weight:bold;"

	styles["string"] = color(theme.Base0B)
	styles["markup.raw"] = color(theme.Base0B)
	styles["diff.plus"] = color(theme.Base0B)

	styles["constant.character.escape"] = color(theme.Base0C)
	styles["markup.quote"] = color(theme.Base0C)

	styles["function"] = color(theme.Base0D)
	styles["function.method"] = color(theme.Base0D)
	styles["markup.heading"] = color(theme.Base0D)

	styles["keyword"] = color(theme.Base0E)
	styles["keyword.storage"] = color(theme.Base0E)
	styles["markup.italic"] = color(theme.Base0E) + "font-style:italic;"
	styles["diff.delta"] = color(theme.Base0E)

	return NewTheme(theme.Scheme, theme.ColorScheme, UITheme{
		StatusBarColor:                 "#" + theme.Base04,
		StatusBarBackgroundColor:       "#" + theme.Base01,
		StatusBarActiveBackgroundColor: "#" + theme.Base02,
	}, html.Theme{
		TabSize:                      4,
		CodeColor:                    "#" + theme.Base05,
		CodeBackgroundColor:          "#" + theme.Base00,
		LineNumberColor:              "#" + theme.Base06,
		LineNumberBackgroundColor:    "#" + theme.Base02,
		HighlightColor:               "#" + theme.Base07,
		SymbolsColor:                 "#" + theme.Base06,
		SymbolsBackgroundColor:       "#" + theme.Base02,
		SymbolsActiveBackgroundColor: "#" + theme.Base03,
		SymbolKindBackgroundColor:    "#" + theme.Base00,
		CodeStyles:                   styles,
	}), nil
}

func color(c string) string {
	if !strings.HasPrefix(c, "#") {
		c = "#" + c
	}

	return "color:" + c + ";"
}
