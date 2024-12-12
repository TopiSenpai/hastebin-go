package server

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/topi314/chroma/v2"
	"github.com/topi314/chroma/v2/formatters"
	"github.com/topi314/chroma/v2/lexers"
	"go.gopad.dev/go-tree-sitter-highlight/html"

	"github.com/topi314/gobin/v2/server/database"
)

func (s *Server) formatFile(file database.File, renderer *html.Renderer, theme html.Theme) (string, error) {
	if renderer == nil {
		return file.Content, nil
	}
	lexer := lexers.Get(file.Language)
	if s.cfg.MaxHighlightSize > 0 && len([]rune(file.Content)) > s.cfg.MaxHighlightSize {
		lexer = lexers.Get("plaintext")
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}

	iterator, err := lexer.Tokenise(nil, file.Content)
	if err != nil {
		return "", fmt.Errorf("tokenise: %w", err)
	}

	buff := new(bytes.Buffer)
	if err = renderer.Format(buff, style, iterator); err != nil {
		return "", fmt.Errorf("format: %w", err)
	}

	return buff.String(), nil
}
