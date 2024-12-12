package server

import (
	"bytes"
	"context"
	"fmt"

	"go.gopad.dev/go-tree-sitter-highlight/folds"
	"go.gopad.dev/go-tree-sitter-highlight/highlight"
	"go.gopad.dev/go-tree-sitter-highlight/html"
	"go.gopad.dev/go-tree-sitter-highlight/tags"

	"github.com/topi314/gobin/v2/server/database"
)

func (s *Server) formatFile(file database.File, renderer *html.Renderer, theme Theme) (string, error) {
	if renderer == nil {
		return file.Content, nil
	}

	language := getLanguage(file.Language)

	if s.cfg.MaxHighlightSize > 0 && len([]rune(file.Content)) > s.cfg.MaxHighlightSize {
		language = getLanguage("plaintext")
	}

	ctx := context.Background()

	highlighter := highlight.New()
	events, err := highlighter.Highlight(ctx, language.Highlight, []byte(file.Content), nil)
	if err != nil {
		return "", fmt.Errorf("highlight: %w", err)
	}

	tagsContext := tags.New()
	allTags, _, err := tagsContext.Tags(ctx, language.Tags, []byte(file.Content))
	if err != nil {
		return "", fmt.Errorf("tags: %w", err)
	}
	resolvedTags, err := renderer.ResolveRefs(allTags, []byte(file.Content), language.Tags.SyntaxTypeNames())
	if err != nil {
		return "", fmt.Errorf("resolve refs: %w", err)
	}

	foldsContext := folds.New()
	foldsIter, err := foldsContext.Folds(ctx, language.Folds, []byte(file.Content))
	if err != nil {
		return "", fmt.Errorf("folds: %w", err)
	}

	buff := new(bytes.Buffer)
	if err = renderer.Render(buff, events, resolvedTags, foldsIter, []byte(file.Content), theme.CaptureNames); err != nil {
		return "", fmt.Errorf("render: %w", err)
	}

	return buff.String(), nil
}
