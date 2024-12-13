package cmd

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/topi314/gobin/v2/internal/cfg"
	"github.com/topi314/gobin/v2/internal/ezhttp"
	"github.com/topi314/gobin/v2/server"
)

func NewPostCmd(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:     "post",
		GroupID: "actions",
		Short:   "Posts a document to the gobin server",
		Example: `gobin post "hello world!"
		
Will post "hello world!" to the server`,
		Args:              cobra.ArbitraryArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := viper.BindPFlag("server", cmd.Flags().Lookup("server")); err != nil {
				return err
			}
			if err := viper.BindPFlag("files", cmd.Flags().Lookup("files")); err != nil {
				return err
			}
			if err := viper.BindPFlag("document", cmd.Flags().Lookup("document")); err != nil {
				return err
			}
			if err := viper.BindPFlag("token", cmd.Flags().Lookup("token")); err != nil {
				return err
			}
			return viper.BindPFlag("languages", cmd.Flags().Lookup("languages"))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			files := viper.GetStringSlice("files")
			documentID := viper.GetString("document")
			token := viper.GetString("token")
			languages := viper.GetStringSlice("languages")

			var (
				readers []io.Reader
			)
			if len(files) > 0 {
				for _, file := range files {
					fr, err := os.Open(strings.TrimSpace(file))
					if err != nil {
						return fmt.Errorf("failed to open document file: %w", err)
					}
					readers = append(readers, fr)
				}
			} else {
				info, err := os.Stdin.Stat()
				if err != nil {
					return fmt.Errorf("failed to get stdin info: %w", err)
				}

				if info.Mode()&os.ModeNamedPipe != 0 {
					readers = append(readers, os.Stdin)
				}
			}
			defer func() {
				for _, r := range readers {
					if rc, ok := r.(io.Closer); ok {
						_ = rc.Close()
					}
				}
			}()

			if len(readers) == 0 {
				if len(args) == 0 {
					return fmt.Errorf("no document provided")
				}
				if len(args) == 1 {
					readers = append(readers, bytes.NewReader([]byte(args[0])))
				} else {
					for _, arg := range args {
						readers = append(readers, bytes.NewReader([]byte(arg)))
					}
				}
			}

			var r io.Reader
			if len(readers) == 0 {
				contentType := ezhttp.DefaultContentTyp
				if len(languages) > 0 {
					contentType = languages[0]
				}
				fileName := "untitiled"
				if file, ok := r.(*os.File); ok {
					fileName = file.Name()
				}
				r = ezhttp.NewHeaderReader(readers[0], http.Header{
					ezhttp.HeaderContentType: []string{
						mime.FormatMediaType(contentType, map[string]string{
							"filename": fileName,
						}),
					},
				})

			} else {
				buff := new(bytes.Buffer)
				mpw := multipart.NewWriter(buff)

				for i, rr := range readers {
					contentType := ezhttp.DefaultContentTyp
					if len(languages) > i {
						contentType = languages[i]
					}
					fileName := fmt.Sprintf("untitiled%d", i)
					if file, ok := rr.(*os.File); ok {
						fileName = file.Name()
					}
					part, err := mpw.CreatePart(textproto.MIMEHeader{
						ezhttp.HeaderContentDisposition: []string{
							mime.FormatMediaType("form-data", map[string]string{
								"name":     fmt.Sprintf("file-%d", i),
								"filename": fileName,
							}),
						},
						ezhttp.HeaderContentType: []string{contentType},
					})
					if err != nil {
						return fmt.Errorf("failed to create multipart part")
					}
					if _, err = io.Copy(part, rr); err != nil {
						return fmt.Errorf("failed to write multipart part")
					}
				}

				if err := mpw.Close(); err != nil {
					return fmt.Errorf("failed to close multipart writer")
				}
				r = ezhttp.NewHeaderReader(buff, http.Header{
					ezhttp.HeaderContentType: []string{mpw.FormDataContentType()},
				})
			}

			var (
				rs  *http.Response
				err error
			)
			if documentID == "" {
				rs, err = ezhttp.Post("/documents", r)
				if err != nil {
					return fmt.Errorf("failed to create document: %w", err)
				}
			} else {
				if token == "" {
					token = viper.GetString("tokens_" + documentID)
				}
				if token == "" {
					return fmt.Errorf("no token found or provided for document: %s", documentID)
				}
				rs, err = ezhttp.Patch("/documents/"+documentID, token, r)
				if err != nil {
					return fmt.Errorf("failed to update document: %w", err)
				}
			}
			defer func() {
				_ = rs.Body.Close()
			}()

			var documentRs server.DocumentResponse
			if err = ezhttp.ProcessBody("post document", rs, &documentRs); err != nil {
				return fmt.Errorf("failed to process response: %w", err)
			}

			method := "Updated"
			if documentID == "" {
				method = "Created"
			}
			cmd.Printf("%s document with ID: %s, Version: %d, URL: %s/%s\n", method, documentRs.Key, documentRs.Version, viper.GetString("server"), documentRs.Key)

			if documentID != "" {
				return nil
			}

			path, err := cfg.Update(func(m map[string]string) {
				m["TOKENS_"+documentRs.Key] = documentRs.Token
			})
			if err != nil {
				return fmt.Errorf("failed to update config: %w", err)
			}
			cmd.Println("Saved token to:", path)
			return nil
		},
	}

	parent.AddCommand(cmd)

	cmd.Flags().StringP("server", "s", "", "Gobin server address")
	cmd.Flags().StringSliceP("files", "f", nil, "The files to post")
	cmd.Flags().StringP("document", "d", "", "The document to update")
	cmd.Flags().StringP("token", "t", "", "The token for the document to update")
	cmd.Flags().StringP("languages", "l", "", "The language of the documents")

	if err := cmd.RegisterFlagCompletionFunc("files", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveDefault
	}); err != nil {
		log.Printf("failed to register files flag completion func: %s", err)
	}

	if err := cmd.RegisterFlagCompletionFunc("document", documentCompletion); err != nil {
		log.Printf("failed to register document flag completion func: %s", err)
	}
}
