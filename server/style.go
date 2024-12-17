package server

import (
	"strings"
)

type rawStyle struct {
	Text       string `toml:"text"`
	Background string `toml:"background"`
	Bold       bool   `toml:"bold"`
	Italic     bool   `toml:"italic"`
	Underline  bool   `toml:"underline"`
}

func (s rawStyle) toCSS(colors map[string]string) string {
	text := getColor(colors, s.Text)
	background := getColor(colors, s.Background)

	var cssStyle string
	if text != "" {
		cssStyle += "color:" + text + ";"
	}

	if background != "" {
		cssStyle += "background-color:" + background + ";"
	}

	if s.Bold {
		cssStyle += "font-weight:bold;"
	}

	if s.Italic {
		cssStyle += "font-style:italic;"
	}

	if s.Underline {
		cssStyle += "text-decoration:underline;"
	}

	return cssStyle
}

func getColor(colors map[string]string, color string) string {
	if color == "" {
		return ""
	}

	colorRef, ok := strings.CutPrefix(color, "$")
	if ok {
		if c, ok := colors[colorRef]; ok {
			return c
		}
	}

	return color
}
