package server

import (
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"github.com/pelletier/go-toml/v2"
	"github.com/topi314/chroma/v2"
	"github.com/topi314/chroma/v2/lexers"
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

func getLanguage(name string) Language {
	l, ok := languages[name]
	if !ok {
		return languages["plaintext"]
	}
	return l
}

func findLanguage(language string, contentType string, fileName string, content string) string {
	var lexer chroma.Lexer
	if language != "" {
		lexer = lexers.Get(language)
	}
	if lexer != nil {
		return lexer.Config().Name
	}

	if contentType != "" && contentType != "application/octet-stream" {
		lexer = lexers.MatchMimeType(contentType)
	}
	if lexer != nil {
		return lexer.Config().Name
	}

	if contentType != "" {
		lexer = lexers.Get(contentType)
	}
	if lexer != nil {
		return lexer.Config().Name
	}

	if fileName != "" {
		lexer = lexers.Match(fileName)
	}
	if lexer != nil {
		return lexer.Config().Name
	}

	if len(content) > 0 {
		lexer = lexers.Analyse(content)
	}
	if lexer != nil {
		return lexer.Config().Name
	}

	return "plaintext"
}

type languageConfigs struct {
	Languages []LanguageConfig `toml:"languages"`
}

type LanguageConfig struct {
	Name              string   `toml:"name"`
	AltNames          []string `toml:"alt_names"`
	MimeTypes         []string `toml:"mime_types"`
	FileTypes         []string `toml:"file_types"`
	Files             []string `toml:"files"`
	GrammarSymbolName string   `toml:"grammar_symbol_name"`
}

func LoadLanguages(data []byte) error {
	var configs languageConfigs
	if err := toml.Unmarshal(data, &configs); err != nil {
		return err
	}

	for _, cfg := range configs.Languages {

	}

	return nil
}

func loadLanguage(cfg LanguageConfig) (*Language, error) {
	libPath := filepath.Join("grammars", cfg.Name)
	if _, err := os.Stat(libPath); err != nil {
		return nil, err
	}


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