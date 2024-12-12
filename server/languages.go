package server

import (
	"github.com/tree-sitter/go-tree-sitter"
	"go.gopad.dev/go-tree-sitter-highlight/folds"
	"go.gopad.dev/go-tree-sitter-highlight/highlight"
	"go.gopad.dev/go-tree-sitter-highlight/tags"
)

var languages = make(map[string]Language)

func registerLanguage(name string, l Language) {
	languages[name] = l
}

type Language struct {
	Language  *tree_sitter.Language
	Highlight highlight.Configuration
	Folds     folds.Configuration
	Tags      tags.Configuration
}
