package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/go-chi/chi/v5"
	"github.com/topi314/tint"

	"github.com/topi314/gobin/v2/internal/ezhttp"
	"github.com/topi314/gobin/v2/internal/flags"
	"github.com/topi314/gobin/v2/internal/gio"
	"github.com/topi314/gobin/v2/internal/httperr"
	"github.com/topi314/gobin/v2/server/database"
	"github.com/topi314/gobin/v2/server/templates"
)

var (
	ErrInvalidMultipartPartName   = errors.New("invalid multipart part name")
	ErrInvalidDocumentFileName    = errors.New("invalid document file name")
	ErrInvalidDocumentFileContent = errors.New("invalid document file content")
	ErrDuplicateDocumentFileNames = errors.New("duplicate document file names")
	ErrDocumentTooLarge           = func(maxLength int64) error {
		return fmt.Errorf("document too large, must be less than %d chars", maxLength)
	}
	ErrInvalidExpiresAt = errors.New("invalid expires_at, must be in the future")
)

var VersionTimeFormat = "2006-01-02 15:04:05"

type (
	DocumentResponse struct {
		Key          string         `json:"key"`
		Version      int64          `json:"version"`
		VersionLabel string         `json:"version_label,omitempty"`
		VersionTime  string         `json:"version_time,omitempty"`
		Files        []ResponseFile `json:"files"`
		Token        string         `json:"token,omitempty"`
	}

	ResponseFile struct {
		Name      string     `json:"name"`
		Content   string     `json:"content,omitempty"`
		Formatted string     `json:"formatted,omitempty"`
		Language  string     `json:"language"`
		ExpiresAt *time.Time `json:"expires_at"`
	}

	RequestFile struct {
		Name      string
		Content   string
		Language  string
		ExpiresAt *time.Time
	}

	DeleteResponse struct {
		Versions int `json:"versions"`
	}

	ShareRequest struct {
		Permissions []string `json:"permissions"`
	}

	ShareResponse struct {
		Token string `json:"token"`
	}
)

func (s *Server) DocumentVersions(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "documentID")
	withContent := r.URL.Query().Get("withContent") == "true"

	versions, err := s.db.GetDocumentVersionsWithFiles(r.Context(), documentID, withContent)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.error(w, r, httperr.NotFound(err))
			return
		}
		s.error(w, r, fmt.Errorf("failed to get document versions: %w", err))
		return
	}

	theme := getTheme(r)
	folds := getFolds(r)

	var response []DocumentResponse
	for version, dbFiles := range versions {
		files := make([]ResponseFile, len(dbFiles))
		for i, file := range dbFiles {
			var formatted string
			if withContent {
				formatted, err = s.formatFile(r.Context(), file, s.htmlRenderer, theme, folds)
				if err != nil {
					s.error(w, r, err)
					return
				}
			}

			files[i] = ResponseFile{
				Name:      file.Name,
				Content:   file.Content,
				Formatted: formatted,
				Language:  file.Language,
				ExpiresAt: file.ExpiresAt,
			}
		}
		response = append(response, DocumentResponse{
			Key:     documentID,
			Version: version,
			Files:   nil,
		})
	}

	s.ok(w, r, response)
}

func (s *Server) GetPrettyDocument(w http.ResponseWriter, r *http.Request) {
	document, err := s.getDocument(r, func(documentID string) string {
		uri := new(url.URL)
		*uri = *r.URL
		uri.Path = fmt.Sprintf("/%s", documentID)
		return uri.String()
	})
	if err != nil {
		if !errors.Is(err, ErrDocumentNotFound) {
			s.prettyError(w, r, err)
			return
		}
		if r.URL.Path != "/" {
			s.redirectRoot(w, r)
			return
		}
	}

	if document == nil {
		document = &database.Document{
			Files: []database.File{{
				Name:     "untitled",
				Content:  "",
				Language: "auto",
			}},
		}
	}

	versions, err := s.db.GetDocumentVersions(r.Context(), document.ID)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		s.prettyError(w, r, fmt.Errorf("failed to get document versions: %w", err))
		return
	}

	theme := getTheme(r)
	folds := getFolds(r)
	fileName := r.URL.Query().Get("file")

	var (
		currentFile int
		totalLength int
	)
	templateFiles := make([]templates.File, len(document.Files))
	for i, file := range document.Files {
		formatted, err := s.formatFile(r.Context(), file, s.htmlRenderer, theme, folds)
		if err != nil {
			s.prettyError(w, r, err)
			return
		}
		if strings.EqualFold(file.Name, fileName) {
			currentFile = i
		}
		templateFiles[i] = templates.File{
			Name:      file.Name,
			Content:   file.Content,
			Formatted: formatted,
			Language:  file.Language,
		}
		totalLength += len([]rune(file.Content))
	}

	templateVersions := make([]templates.DocumentVersion, len(versions))
	for i, v := range versions {
		versionTime := time.UnixMilli(v)
		versionLabel := humanize.Time(versionTime)
		if i == 0 {
			versionLabel += " (current)"
		} else if i == len(versions)-1 {
			versionLabel += " (original)"
		}
		templateVersions[i] = templates.DocumentVersion{
			Version: v,
			Label:   versionLabel,
			Time:    versionTime.Format(VersionTimeFormat),
		}
	}

	var (
		previewURL string
		previewAlt string
	)
	if s.cfg.Preview != nil {
		previewURL = "https://" + r.Host + "/" + document.ID
		if version := chi.URLParam(r, "version"); version != "" {
			previewURL += "/" + version
		}
		previewURL += "/preview"
		if r.URL.RawQuery != "" {
			previewURL += "?" + r.URL.RawQuery
		}

		previewAlt = s.shortContent(templateFiles[currentFile].Content)
	}
	if err = templates.Document(templates.DocumentVars{
		ID:      document.ID,
		Version: document.Version,
		Edit:    document.ID == "",

		Files:       templateFiles,
		CurrentFile: currentFile,
		TotalLength: totalLength,
		Versions:    templateVersions,

		Languages:   getLanguageNames(),
		Themes:      s.themes,
		Theme:       theme.Name,
		ColorScheme: theme.ColorScheme,
		Folds:       folds,

		Max:        s.cfg.MaxDocumentSize,
		Host:       r.Host,
		PreviewURL: previewURL,
		PreviewAlt: previewAlt,
	}).Render(r.Context(), w); err != nil {
		slog.ErrorContext(r.Context(), "failed to execute template", tint.Err(err))
	}
}

func (s *Server) GetDocument(w http.ResponseWriter, r *http.Request) {
	document, err := s.getDocument(r, nil)
	if err != nil {
		s.error(w, r, err)
		return
	}

	theme := getTheme(r)
	folds := getFolds(r)
	fileName := r.URL.Query().Get("file")

	if fileName != "" {
		for _, file := range document.Files {
			if strings.EqualFold(file.Name, fileName) {
				if language := r.URL.Query().Get("language"); language != "" {
					file.Language = getLanguageFallback(language).Config.Name
				}

				formatted, err := s.formatFile(r.Context(), file, s.htmlRenderer, theme, folds)
				if err != nil {
					s.error(w, r, err)
					return
				}
				s.ok(w, r, ResponseFile{
					Name:      file.Name,
					Content:   file.Content,
					Formatted: formatted,
					Language:  file.Language,
				})
				return
			}
		}
		s.error(w, r, httperr.NotFound(ErrDocumentFileNotFound))
		return
	}

	response := DocumentResponse{
		Key:     document.ID,
		Version: document.Version,
		Files:   make([]ResponseFile, len(document.Files)),
	}
	for i, file := range document.Files {
		formatted, err := s.formatFile(r.Context(), file, s.htmlRenderer, theme, folds)
		if err != nil {
			s.error(w, r, err)
			return
		}
		response.Files[i] = ResponseFile{
			Name:      file.Name,
			Content:   file.Content,
			Formatted: formatted,
			Language:  file.Language,
		}
	}

	s.ok(w, r, response)
}

func (s *Server) GetRawDocument(w http.ResponseWriter, r *http.Request) {
	document, err := s.getDocument(r, nil)
	if err != nil {
		s.error(w, r, err)
		return
	}

	theme := getTheme(r)
	folds := getFolds(r)

	if len(document.Files) == 1 {
		file := document.Files[0]

		formatted, err := s.formatFile(r.Context(), file, s.htmlRenderer, theme, folds)
		if err != nil {
			s.error(w, r, fmt.Errorf("failed to render raw document: %w", err))
			return
		}

		contentType := ezhttp.ContentTypeHTML
		fileName := file.Name + ".html"

		w.Header().Set(ezhttp.HeaderContentDisposition, mime.FormatMediaType("inline", map[string]string{
			"name":     fileName,
			"filename": fileName,
		}))

		language := getLanguageFallback(file.Language)
		w.Header().Set(ezhttp.HeaderLanguage, language.Config.Name)

		w.Header().Set(ezhttp.HeaderContentType, contentType)
		if _, err = w.Write([]byte(formatted)); err != nil {
			s.error(w, r, err)
		}
		return
	}

	mpw := multipart.NewWriter(w)
	for i, file := range document.Files {
		formatted, err := s.formatFile(r.Context(), file, s.htmlRenderer, theme, folds)
		if err != nil {
			s.error(w, r, fmt.Errorf("failed to render raw document: %w", err))
			return
		}

		headers := make(textproto.MIMEHeader, 2)
		headers.Set(ezhttp.HeaderContentDisposition, mime.FormatMediaType("form-data", map[string]string{
			"name":     fmt.Sprintf("file-%d", i),
			"filename": file.Name,
		}))

		language := getLanguageFallback(file.Language)

		headers.Set(ezhttp.HeaderLanguage, language.Highlight.LanguageName)
		headers.Set(ezhttp.HeaderContentType, ezhttp.ContentTypeHTML)

		part, err := mpw.CreatePart(headers)
		if err != nil {
			s.error(w, r, err)
			return
		}
		if _, err = part.Write([]byte(formatted + "\n")); err != nil {
			s.error(w, r, err)
			return
		}
	}

	if err = mpw.Close(); err != nil {
		s.error(w, r, err)
		return
	}
}

func (s *Server) GetDocumentPreview(w http.ResponseWriter, r *http.Request) {
	//document, err := s.getDocument(r, func(documentID string) string {
	//	uri := new(url.URL)
	//	*uri = *r.URL
	//	uri.Path = fmt.Sprintf("/%s/preview", documentID)
	//	return uri.String()
	//})
	//if err != nil {
	//	s.error(w, r, err)
	//	return
	//}
	//
	//theme := getTheme(r)
	//fileName := r.URL.Query().Get("file")
	//
	//var currentFile int
	//for i, file := range document.Files {
	//	if file.Name == fileName {
	//		currentFile = i
	//		break
	//	}
	//}
	//
	//file := document.Files[currentFile]
	//file.Content = s.shortContent(file.Content)
	//
	//formatted, err := s.formatFile(file, formatter, style)
	//if err != nil {
	//	s.prettyError(w, r, fmt.Errorf("failed to render document preview: %w", err))
	//	return
	//}
	//
	//png, err := s.convertSVG2PNG(r.Context(), formatted)
	//if err != nil {
	//	s.error(w, r, fmt.Errorf("failed to convert document preview: %w", err))
	//	return
	//}
	//
	//w.Header().Set(ezhttp.HeaderContentType, ezhttp.ContentTypePNG)
	//if r.Method == http.MethodHead {
	//	w.Header().Set(ezhttp.HeaderContentLength, strconv.Itoa(len(png)))
	//	w.WriteHeader(http.StatusOK)
	//	return
	//}
	//_, _ = w.Write(png)
}

func (s *Server) getDocument(r *http.Request, fallbackURL func(documentID string) string) (*database.Document, error) {
	documentID := chi.URLParam(r, "documentID")
	if i := strings.Index(documentID, "."); i > 0 {
		documentID = documentID[:i]
	}
	if documentID == "" {
		return nil, httperr.NotFound(ErrDocumentNotFound)
	}

	var version int64
	if versionStr := chi.URLParam(r, "version"); versionStr != "" {
		var err error
		version, err = strconv.ParseInt(versionStr, 10, 64)
		if err != nil {
			return nil, httperr.BadRequest(ErrInvalidDocumentVersion)
		}
	}

	var (
		files []database.File
		err   error
	)
	if version == 0 {
		files, err = s.db.GetDocument(r.Context(), documentID)
	} else {
		files, err = s.db.GetDocumentVersion(r.Context(), documentID, version)
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if fallbackURL != nil && version > 0 {
				return nil, httperr.Found(fallbackURL(documentID))
			}
			return nil, httperr.NotFound(ErrDocumentNotFound)
		}
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	return &database.Document{
		ID:      documentID,
		Version: version,
		Files:   files,
	}, nil
}

func (s *Server) GetDocumentFile(w http.ResponseWriter, r *http.Request) {
	file, err := s.getDocumentFile(r)
	if err != nil {
		s.error(w, r, err)
		return
	}

	theme := getTheme(r)
	folds := getFolds(r)

	if language := r.URL.Query().Get("language"); language != "" {
		file.Language = getLanguageFallback(language).Config.Name
	}

	formatted, err := s.formatFile(r.Context(), *file, s.htmlRenderer, theme, folds)
	if err != nil {
		s.error(w, r, err)
		return
	}

	s.ok(w, r, ResponseFile{
		Name:      file.Name,
		Content:   file.Content,
		Formatted: formatted,
		Language:  file.Language,
	})
}

func (s *Server) GetRawDocumentFile(w http.ResponseWriter, r *http.Request) {
	file, err := s.getDocumentFile(r)
	if err != nil {
		s.error(w, r, err)
		return
	}

	theme := getTheme(r)
	folds := getFolds(r)

	language := getLanguageFallback(file.Language)
	w.Header().Set(ezhttp.HeaderLanguage, language.Config.Name)

	formatted, err := s.formatFile(r.Context(), *file, s.htmlRenderer, theme, folds)
	if err != nil {
		s.error(w, r, fmt.Errorf("failed to render raw document: %w", err))
		return
	}

	contentType := "text/html; charset=UTF-8"
	fileName := file.Name + ".html"

	w.Header().Set(ezhttp.HeaderContentDisposition, mime.FormatMediaType("inline", map[string]string{
		"name":     fileName,
		"filename": fileName,
	}))
	w.Header().Set(ezhttp.HeaderContentType, contentType)

	if _, err = w.Write([]byte(formatted)); err != nil {
		s.error(w, r, err)
		return
	}
}

func (s *Server) getDocumentFile(r *http.Request) (*database.File, error) {
	documentID := chi.URLParam(r, "documentID")
	if i := strings.Index(documentID, "."); i > 0 {
		documentID = documentID[:i]
	}
	if documentID == "" {
		return nil, httperr.NotFound(ErrDocumentFileNotFound)
	}

	versionStr := chi.URLParam(r, "version")
	var version int64
	if versionStr != "" {
		var err error
		version, err = strconv.ParseInt(versionStr, 10, 64)
		if err != nil {
			return nil, httperr.BadRequest(ErrInvalidDocumentVersion)
		}
	}

	fileName := chi.URLParam(r, "fileName")
	if fileName == "" {
		return nil, httperr.NotFound(ErrDocumentFileNotFound)
	}

	var (
		file *database.File
		err  error
	)
	if version == 0 {
		file, err = s.db.GetDocumentFile(r.Context(), documentID, fileName)
	} else {
		file, err = s.db.GetDocumentFileVersion(r.Context(), documentID, version, fileName)
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, httperr.NotFound(ErrDocumentFileNotFound)
		}
		return nil, fmt.Errorf("failed to get document file: %w", err)
	}

	return file, nil
}

func (s *Server) PostDocument(w http.ResponseWriter, r *http.Request) {
	files, err := s.parseDocumentFiles(r)
	if err != nil {
		s.error(w, r, err)
		return
	}

	var dbFiles []database.File
	for i, file := range files {
		dbFiles = append(dbFiles, database.File{
			Name:       file.Name,
			Content:    file.Content,
			Language:   file.Language,
			ExpiresAt:  file.ExpiresAt,
			OrderIndex: i,
		})
	}

	documentID, version, err := s.db.CreateDocument(r.Context(), dbFiles)
	if err != nil {
		s.error(w, r, fmt.Errorf("failed to create document: %w", err))
		return
	}

	theme := getTheme(r)
	folds := getFolds(r)

	var rsFiles []ResponseFile
	for _, file := range dbFiles {
		formatted, err := s.formatFile(r.Context(), file, s.htmlRenderer, theme, folds)
		if err != nil {
			s.error(w, r, err)
			return
		}
		rsFiles = append(rsFiles, ResponseFile{
			Name:      file.Name,
			Content:   file.Content,
			Formatted: formatted,
			Language:  file.Language,
			ExpiresAt: file.ExpiresAt,
		})
	}

	token, err := s.NewToken(*documentID, AllPermissions)
	if err != nil {
		s.error(w, r, fmt.Errorf("failed to create jwt token: %w", err))
		return
	}

	versionTime := time.UnixMilli(*version)
	s.json(w, r, DocumentResponse{
		Key:          *documentID,
		Version:      *version,
		VersionLabel: humanize.Time(versionTime) + " (original)",
		VersionTime:  versionTime.Format(VersionTimeFormat),
		Files:        rsFiles,
		Token:        token,
	}, http.StatusCreated)

}

func (s *Server) PatchDocument(w http.ResponseWriter, r *http.Request) {
	files, err := s.parseDocumentFiles(r)
	if err != nil {
		s.error(w, r, err)
		return
	}

	claims := GetClaims(r)
	if flags.Misses(claims.Permissions, PermissionWrite) {
		s.error(w, r, httperr.Forbidden(ErrPermissionDenied("write")))
		return
	}

	documentID := chi.URLParam(r, "documentID")

	var dbFiles []database.File
	for i, file := range files {
		dbFiles = append(dbFiles, database.File{
			Name:       file.Name,
			Content:    file.Content,
			Language:   file.Language,
			ExpiresAt:  file.ExpiresAt,
			OrderIndex: i,
		})
	}

	version, err := s.db.UpdateDocument(r.Context(), documentID, dbFiles)
	if err != nil {
		s.error(w, r, fmt.Errorf("failed to update document: %w", err))
		return
	}

	theme := getTheme(r)
	folds := getFolds(r)

	var rsFiles []ResponseFile
	for _, file := range dbFiles {
		formatted, err := s.formatFile(r.Context(), file, s.htmlRenderer, theme, folds)
		if err != nil {
			s.error(w, r, err)
			return
		}
		rsFiles = append(rsFiles, ResponseFile{
			Name:      file.Name,
			Content:   file.Content,
			Formatted: formatted,
			Language:  file.Language,
			ExpiresAt: file.ExpiresAt,
		})
	}

	webhooksFiles := make([]WebhookDocumentFile, len(files))
	for i, file := range files {
		webhooksFiles[i] = WebhookDocumentFile{
			Name:      file.Name,
			Content:   file.Content,
			Language:  file.Language,
			ExpiresAt: file.ExpiresAt,
		}
	}
	s.ExecuteWebhooks(r.Context(), WebhookEventUpdate, WebhookDocument{
		Key:     documentID,
		Version: *version,
		Files:   webhooksFiles,
	})

	versionTime := time.UnixMilli(*version)
	s.json(w, r, DocumentResponse{
		Key:          documentID,
		Version:      *version,
		VersionLabel: humanize.Time(versionTime) + " (current)",
		VersionTime:  versionTime.Format(VersionTimeFormat),
		Files:        rsFiles,
	}, http.StatusOK)
}

func (s *Server) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r)
	if flags.Misses(claims.Permissions, PermissionDelete) {
		s.error(w, r, httperr.Forbidden(ErrPermissionDenied("webhook")))
		return
	}

	documentID := chi.URLParam(r, "documentID")
	var version int64
	if versionStr := chi.URLParam(r, "version"); versionStr != "" {
		var err error
		version, err = strconv.ParseInt(versionStr, 10, 64)
		if err != nil {
			s.error(w, r, httperr.BadRequest(ErrInvalidDocumentVersion))
			return
		}
	}

	var (
		document *database.Document
		err      error
	)
	if version == 0 {
		document, err = s.db.DeleteDocument(r.Context(), documentID)
	} else {
		document, err = s.db.DeleteDocumentVersion(r.Context(), documentID, version)
	}
	if err != nil {
		s.error(w, r, fmt.Errorf("failed to delete document: %w", err))
		return
	}

	webhooksFiles := make([]WebhookDocumentFile, len(document.Files))
	for i, file := range document.Files {
		webhooksFiles[i] = WebhookDocumentFile{
			Name:      file.Name,
			Content:   file.Content,
			Language:  file.Language,
			ExpiresAt: file.ExpiresAt,
		}
	}
	s.ExecuteWebhooks(r.Context(), WebhookEventDelete, WebhookDocument{
		Key:     document.ID,
		Version: document.Version,
		Files:   webhooksFiles,
	})

	if version == 0 {
		s.ok(w, r, nil)
	}

	count, err := s.db.GetVersionCount(r.Context(), documentID)
	if err != nil {
		s.error(w, r, err)
		return
	}
	s.ok(w, r, DeleteResponse{
		Versions: count,
	})
}

func (s *Server) PostDocumentShare(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "documentID")

	var shareRequest ShareRequest
	if err := json.NewDecoder(r.Body).Decode(&shareRequest); err != nil {
		s.error(w, r, httperr.BadRequest(err))
		return
	}

	if len(shareRequest.Permissions) == 0 {
		s.error(w, r, httperr.BadRequest(ErrNoPermissions))
		return
	}

	for _, permission := range shareRequest.Permissions {
		if !slices.Contains(AllStringPermissions, permission) {
			s.error(w, r, httperr.BadRequest(ErrUnknownPermission(permission)))
			return
		}
	}

	claims := GetClaims(r)
	if claims.Subject != documentID || flags.Misses(claims.Permissions, PermissionShare) {
		s.error(w, r, httperr.Forbidden(ErrPermissionDenied("share")))
		return
	}

	perms, err := parsePermissions(claims.Permissions, shareRequest.Permissions)
	if err != nil {
		s.error(w, r, httperr.Forbidden(err))
		return
	}

	token, err := s.NewToken(documentID, perms)
	if err != nil {
		s.error(w, r, fmt.Errorf("failed to create new token: %w", err))
		return
	}

	s.ok(w, r, ShareResponse{Token: token})
}

func (s *Server) parseDocumentFiles(r *http.Request) ([]RequestFile, error) {
	var files []RequestFile
	contentType := r.Header.Get(ezhttp.HeaderContentType)
	if contentType != "" {
		var err error
		contentType, _, err = mime.ParseMediaType(contentType)
		if err != nil {
			return nil, fmt.Errorf("failed to parse content type: %w", err)
		}
	}
	query := r.URL.Query()

	expiresAt, err := getExpiresAt(query, r.Header)
	if err != nil {
		return nil, err
	}

	if contentType == "multipart/form-data" {
		mr, err := r.MultipartReader()
		if err != nil {
			return nil, fmt.Errorf("failed to get multipart reader: %w", err)
		}

		var limitReader *gio.LimitedReader
		if s.cfg.MaxDocumentSize > 0 {
			limitReader = gio.LimitReader(nil, s.cfg.MaxDocumentSize)
		}

		for i := 0; ; i++ {
			part, err := mr.NextPart()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return nil, fmt.Errorf("failed to get multipart part: %w", err)
			}

			if part.FormName() != fmt.Sprintf("file-%d", i) {
				return nil, httperr.BadRequest(ErrInvalidMultipartPartName)
			}

			if part.FileName() == "" {
				return nil, httperr.BadRequest(ErrInvalidDocumentFileName)
			}

			reader := io.Reader(part)
			if limitReader != nil {
				limitReader.R = part
				reader = limitReader
			}
			data, err := io.ReadAll(reader)
			if err != nil {
				if errors.Is(err, gio.ErrLimitReached) {
					return nil, httperr.BadRequest(ErrDocumentTooLarge(s.cfg.MaxDocumentSize))
				}
				return nil, fmt.Errorf("failed to read part body: %w", err)
			}

			if len(data) == 0 {
				return nil, httperr.BadRequest(ErrInvalidDocumentFileContent)
			}

			partContentType := part.Header.Get(ezhttp.HeaderContentType)
			if partContentType != "" {
				partContentType, _, _ = mime.ParseMediaType(partContentType)
			}

			newExpiresAt, err := getExpiresAt(nil, http.Header(part.Header))
			if err != nil {
				return nil, err
			}
			if newExpiresAt != nil {
				expiresAt = newExpiresAt
			}

			files = append(files, RequestFile{
				Name:      part.FileName(),
				Content:   string(data),
				Language:  findLanguage(part.Header.Get(ezhttp.HeaderLanguage), partContentType, part.FileName(), string(data)),
				ExpiresAt: expiresAt,
			})
		}
	} else {
		reader := io.Reader(r.Body)
		if s.cfg.MaxDocumentSize > 0 {
			reader = gio.LimitReader(r.Body, s.cfg.MaxDocumentSize)
		}

		data, err := io.ReadAll(reader)
		if err != nil {
			if errors.Is(err, gio.ErrLimitReached) {
				return nil, httperr.BadRequest(ErrDocumentTooLarge(s.cfg.MaxDocumentSize))
			}
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}

		params := make(map[string]string)
		if contentDisposition := r.Header.Get(ezhttp.HeaderContentDisposition); contentDisposition != "" {
			_, params, err = mime.ParseMediaType(contentDisposition)
			if err != nil {
				return nil, fmt.Errorf("failed to parse content disposition: %w", err)
			}
		}

		name := params["filename"]
		if name == "" {
			name = "untitled"
		}

		language := query.Get("language")
		if language == "" {
			language = r.Header.Get(ezhttp.HeaderLanguage)
		}

		files = []RequestFile{{
			Name:      name,
			Content:   string(data),
			Language:  findLanguage(language, contentType, params["filename"], string(data)),
			ExpiresAt: expiresAt,
		}}
	}
	for i, file := range files {
		for ii, f := range files {
			if strings.EqualFold(file.Name, f.Name) && i != ii {
				return nil, httperr.BadRequest(ErrDuplicateDocumentFileNames)
			}
		}
	}
	return files, nil
}

func getExpiresAt(query url.Values, header http.Header) (*time.Time, error) {
	expiresAtStr := query.Get("expires")
	if expiresAtStr == "" {
		expiresAtStr = header.Get("Expires")
	}
	if expiresAtStr == "" {
		return nil, nil
	}
	expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
	if err != nil {
		return nil, httperr.BadRequest(fmt.Errorf("failed to parse expires_at query param: %w", err))
	}
	if expiresAt.Before(time.Now()) {
		return nil, httperr.BadRequest(ErrInvalidExpiresAt)
	}
	return &expiresAt, nil
}

func getFolds(r *http.Request) bool {
	var folds *bool

	if foldsStr, err := r.Cookie("folds"); err == nil {
		if parsedFolds, err := strconv.ParseBool(foldsStr.Value); err == nil {
			folds = &parsedFolds
		}
	}

	if queryFolds := r.URL.Query().Get("folds"); queryFolds != "" {
		if parsedFolds, err := strconv.ParseBool(queryFolds); err == nil {
			folds = &parsedFolds
		}
	}

	if folds == nil && strings.Contains(r.UserAgent(), "Firefox") {
		return true
	}

	if folds == nil {
		return false
	}

	return *folds
}
