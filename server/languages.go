package server

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/pelletier/go-toml/v2"
	"github.com/tree-sitter/go-tree-sitter"
	"go.gopad.dev/go-tree-sitter-highlight/folds"
	"go.gopad.dev/go-tree-sitter-highlight/highlight"
	"go.gopad.dev/go-tree-sitter-highlight/tags"
)

var languages = make(map[string]Language)

type Language struct {
	Language  *tree_sitter.Language
	Config    LanguageConfig
	Highlight highlight.Configuration
	Folds     folds.Configuration
	Tags      tags.Configuration
}

func registerLanguage(name string, l Language) {
	languages[name] = l
}

func getLanguage(name string) (Language, bool) {
	l, ok := languages[name]
	return l, ok
}

func getLanguageFallback(name string) Language {
	l, ok := languages[name]
	if !ok {
		return languages["plaintext"]
	}
	return l
}

func getLanguageNames() []string {
	names := make([]string, 0, len(languages))
	for name := range languages {
		names = append(names, name)
	}
	return names
}

func findLanguage(language string, contentType string, fileName string, content string) string {
	for _, lang := range languages {
		if lang.Config.Name == language {
			return lang.Config.Name
		}

		if lang.Config.Name == contentType {
			return lang.Config.Name
		}

		if slices.Contains(lang.Config.AltNames, language) {
			return lang.Config.Name
		}

		if slices.Contains(lang.Config.MimeTypes, contentType) {
			return lang.Config.Name
		}

		if slices.Contains(lang.Config.FileTypes, fileName) {
			return lang.Config.Name
		}

		fileType := filepath.Ext(fileName)
		if slices.Contains(lang.Config.FileTypes, fileType) {
			return lang.Config.Name
		}
	}

	return "plaintext"
}

func injectionLanguage(languageName string) *highlight.Configuration {
	lang, ok := getLanguage(languageName)
	if !ok {
		return nil
	}
	return &lang.Highlight
}

type languageConfigs struct {
	Languages map[string]LanguageConfig `toml:"languages"`
}

type LanguageConfig struct {
	Name              string   `toml:"name"`
	AltNames          []string `toml:"alt_names"`
	MimeTypes         []string `toml:"mime_types"`
	FileTypes         []string `toml:"file_types"`
	Files             []string `toml:"files"`
	GrammarSymbolName string   `toml:"grammar_symbol_name"`
}

func LoadLanguages(fs embed.FS, data []byte) error {
	var configs languageConfigs
	if err := toml.Unmarshal(data, &configs); err != nil {
		return err
	}

	for name, cfg := range configs.Languages {
		cfg.Name = name
		lang, err := loadLanguage(fs, cfg)
		if err != nil {
			return fmt.Errorf("failed to load language %q: %w", cfg.Name, err)
		}
		registerLanguage(cfg.Name, *lang)
	}

	return nil
}

func loadLanguage(fs embed.FS, cfg LanguageConfig) (*Language, error) {
	highlightsQuery, err := loadQuery(fs, cfg.Name, "highlights")
	if err != nil {
		return nil, err
	}

	injectionsQuery, err := loadQuery(fs, cfg.Name, "injections")
	if err != nil {
		return nil, err
	}

	localsQuery, err := loadQuery(fs, cfg.Name, "locals")
	if err != nil {
		return nil, err
	}

	foldsQuery, err := loadQuery(fs, cfg.Name, "folds")
	if err != nil {
		return nil, err
	}

	tagsQuery, err := loadQuery(fs, cfg.Name, "tags")
	if err != nil {
		return nil, err
	}

	libName := fmt.Sprintf("tree-sitter-%s.so", cfg.Name)
	language, err := newLanguage(cfg.GrammarSymbolName, filepath.Join("grammars", libName))
	if err != nil {
		return nil, err
	}

	hlCfg, err := highlight.NewConfiguration(language, cfg.Name, highlightsQuery, injectionsQuery, localsQuery)
	if err != nil {
		return nil, err
	}

	foldsCfg, err := folds.NewConfiguration(language, foldsQuery)
	if err != nil {
		return nil, err
	}

	tagsCfg, err := tags.NewConfiguration(language, tagsQuery, localsQuery)
	if err != nil {
		return nil, err
	}

	return &Language{
		Language:  language,
		Config:    cfg,
		Highlight: *hlCfg,
		Folds:     *foldsCfg,
		Tags:      *tagsCfg,
	}, nil
}

func loadQuery(fs embed.FS, languageName string, name string) ([]byte, error) {
	path := filepath.Join("queries", languageName, name+".scm")
	data, err := fs.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	return data, nil
}

func newLanguage(symbolName string, path string) (*tree_sitter.Language, error) {
	lib, err := purego.Dlopen(path, purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		return nil, fmt.Errorf("failed to open language library: %w", err)
	}

	var newTreeSitter func() uintptr
	purego.RegisterLibFunc(&newTreeSitter, lib, "tree_sitter_"+symbolName)

	return tree_sitter.NewLanguage(unsafe.Pointer(newTreeSitter())), nil
}
