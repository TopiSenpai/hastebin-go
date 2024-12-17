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
	Version int64
	Edit    bool

	Files       []File
	CurrentFile int
	TotalLength int
	Versions    []DocumentVersion

	PreviewURL string
	PreviewAlt string

	Languages   []string
	Themes      []Theme
	Theme       string
	ColorScheme string
	Folds       bool
	Max         int64
	Host        string
}

type File struct {
	Name      string `json:"name"`
	Content   string `json:"content"`
	Formatted string `json:"formatted"`
	Language  string `json:"language"`
}

type gobin struct {
	Key         string `json:"key"`
	Version     int64  `json:"version"`
	Mode        string `json:"mode"`
	Files       []File `json:"files"`
	CurrentFile int    `json:"current_file"`
	ExpireIn    int    `json:"expire_in"`
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

func (v DocumentVars) URL() string {
	return "https://" + v.Host
}

func (v DocumentVars) ThemeCSSURL() string {
	return fmt.Sprintf("/assets/theme.css?theme=%s", v.Theme)
}

type DocumentVersion struct {
	Version int64
	Label   string
	Time    string
}

type Theme struct {
	Name        string
	ColorScheme string
}

type ErrorVars struct {
	Error     string
	Status    int
	Path      string
	RequestID string
}
