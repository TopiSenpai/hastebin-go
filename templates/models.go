package templates

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/a-h/templ"
)

func WriteUnsafe(str string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := w.Write([]byte(str))
		return err
	})
}

type DocumentVars struct {
	ID      string
	Version string
	Edit    bool

	Files       []File
	CurrentFile int
	Versions    []DocumentVersion

	Preview    bool
	PreviewAlt string

	Lexers []string
	Styles []Style
	Style  string
	Theme  string
	Max    int
	Host   string
}

type File struct {
	Name      string `json:"name"`
	Content   string `json:"content"`
	Formatted string `json:"formatted"`
	Language  string `json:"language"`
}

type gobin struct {
	Key         string `json:"key"`
	Version     string `json:"version"`
	Mode        string `json:"mode"`
	Files       []File `json:"files"`
	CurrentFile int    `json:"current_file"`
}

func (v DocumentVars) StateJSON() string {
	mode := "edit"
	if !v.Edit {
		mode = "view"
	}
	data, _ := json.Marshal(gobin{
		Key:         v.ID,
		Version:     v.Version,
		Mode:        mode,
		Files:       v.Files,
		CurrentFile: v.CurrentFile,
	})
	return fmt.Sprintf(`<script id="state" type="application/json">%s</script>`, string(data))
}

func (v DocumentVars) FileClasses(i int) string {
	classes := "file"
	if i == v.CurrentFile {
		classes += " selected"
	}
	return classes
}

func (v DocumentVars) FileTabClasses(i int) string {
	classes := "file-tab"
	if i == v.CurrentFile {
		classes += " initial"
	}
	return classes
}

func (v DocumentVars) PreviewURL() string {
	url := "https://" + v.Host + "/" + v.ID
	if v.Version > "0" {
		url += "/" + v.Version
	}
	return url + "/preview"
}

func (v DocumentVars) URL() string {
	return "https://" + v.Host
}

func (v DocumentVars) ThemeCSSURL() string {
	return fmt.Sprintf("/assets/theme.css?style=%s", v.Style)
}

type DocumentVersion struct {
	Version string
	Label   string
	Time    string
}

type Style struct {
	Name  string
	Theme string
}

type ErrorVars struct {
	Error     string
	Status    int
	Path      string
	RequestID string
}
