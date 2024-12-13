package cmd

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/topi314/gobin/v2/internal/ezhttp"
	"github.com/topi314/gobin/v2/server"
)

func NewGetCmd(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:     "get",
		GroupID: "actions",
		Short:   "Gets a document from the gobin server",
		Example: `gobin get jis74978

Will return the document with the id of jis74978.`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: documentCompletion,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := viper.BindPFlag("server", cmd.Flags().Lookup("server")); err != nil {
				return err
			}
			if err := viper.BindPFlag("file", cmd.Flags().Lookup("file")); err != nil {
				return err
			}
			if err := viper.BindPFlag("version", cmd.Flags().Lookup("version")); err != nil {
				return err
			}
			if err := viper.BindPFlag("versions", cmd.Flags().Lookup("versions")); err != nil {
				return err
			}
			if err := viper.BindPFlag("formatter", cmd.Flags().Lookup("formatter")); err != nil {
				return err
			}
			if err := viper.BindPFlag("language", cmd.Flags().Lookup("language")); err != nil {
				return err
			}
			if err := viper.BindPFlag("style", cmd.Flags().Lookup("style")); err != nil {
				return err
			}
			return viper.BindPFlag("output", cmd.Flags().Lookup("output"))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("document id is required")
			}
			documentID := args[0]
			file := viper.GetString("file")
			version := viper.GetString("version")
			versions := viper.GetBool("versions")
			formatter := viper.GetString("formatter")
			language := viper.GetString("language")
			style := viper.GetString("style")
			output := viper.GetString("output")

			if versions {
				rs, err := ezhttp.Get("/documents/" + documentID + "/versions")
				if err != nil {
					return fmt.Errorf("failed to get document versions: %w", err)
				}
				defer func() {
					_ = rs.Body.Close()
				}()

				var documentVersionsRs []server.DocumentResponse
				if err = ezhttp.ProcessBody("get document versions", rs, &documentVersionsRs); err != nil {
					return err
				}

				var documentVersions string
				for _, documentVersion := range documentVersionsRs {
					documentVersions += fmt.Sprintf("%d: %s\n", documentVersion.Version, humanize.Time(time.UnixMilli(documentVersion.Version)))
				}

				cmd.Printf("Document versions(%d):\n%s", len(documentVersions), documentVersions)
				return nil
			}

			uri := "/documents/" + documentID
			if version != "" {
				uri += "/versions/" + version
			}
			query := make(url.Values)
			if formatter != "" {
				query.Add("formatter", formatter)
			}
			if style != "" {
				query.Add("style", style)
			}
			if file != "" {
				query.Add("file", file)
				if style != "" {
					query.Add("language", language)
				}
			}
			if len(query) > 0 {
				uri += "?" + query.Encode()
			}

			rs, err := ezhttp.Get(uri)
			if err != nil {
				return fmt.Errorf("failed to get document: %w", err)
			}
			defer func() {
				_ = rs.Body.Close()
			}()

			if file != "" {
				var fileRs server.ResponseFile
				if err = ezhttp.ProcessBody("get document file", rs, &fileRs); err != nil {
					return err
				}
				content := fileRs.Content
				if formatter != "" {
					content = fileRs.Formatted
				}

				if output == "" {
					cmd.Println(content)
					return nil
				}

				filePath := filepath.Join(output, fileRs.Name)
				documentFile, err := os.Create(filePath)
				if err != nil {
					return fmt.Errorf("failed to create file to write document: %w", err)
				}
				defer func() {
					_ = documentFile.Close()
				}()

				if _, err = documentFile.WriteString(content); err != nil {
					return fmt.Errorf("failed to write document to file: %w", err)
				}
				cmd.Println("Document file saved to:", filePath)
				return nil
			}

			var documentRs server.DocumentResponse
			if err = ezhttp.ProcessBody("get document", rs, &documentRs); err != nil {
				return err
			}

			for _, dFile := range documentRs.Files {
				content := dFile.Content
				if formatter != "" {
					content = dFile.Formatted
				}

				if output == "" {
					if len(documentRs.Files) > 0 {
						cmd.Printf("File: %s", dFile.Name)
					}
					cmd.Println(content)
					return nil
				}

				if err = func() error {
					filePath := filepath.Join(output, dFile.Name)
					documentFile, err := os.Create(filePath)
					if err != nil {
						return fmt.Errorf("failed to create file to write document: %w", err)
					}
					defer func() {
						_ = documentFile.Close()
					}()

					_, err = documentFile.WriteString(content)
					if err != nil {
						return fmt.Errorf("failed to write document to file: %w", err)
					}
					cmd.Println("Document file saved to:", filePath)
					return nil
				}(); err != nil {
					return err
				}
			}

			return nil
		},
	}

	parent.AddCommand(cmd)

	cmd.Flags().StringP("server", "s", "", "Gobin server address")
	cmd.Flags().StringP("file", "f", "", "The document file to get")
	cmd.Flags().StringP("version", "v", "", "The version of the document to get")
	cmd.Flags().BoolP("versions", "", false, "Get all versions of the document")
	cmd.Flags().StringP("formatter", "r", "terminal16m", "Format the document with syntax highlighting (terminal8, terminal16, terminal256, terminal16m, html, html-standalone, svg, or none)")
	cmd.Flags().StringP("language", "l", "", "The language to render the document with (only works in combination with file)")
	cmd.Flags().StringP("style", "", "", "The style to render the document with")
	cmd.Flags().StringP("output", "o", ".", "The folder to save the document to")

	if err := cmd.RegisterFlagCompletionFunc("formatter", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"terminal8", "terminal16", "terminal256", "terminal16m", "html", "html-standalone", "svg", "none"}, cobra.ShellCompDirectiveNoFileComp
	}); err != nil {
		log.Printf("failed to register formatter flag completion func: %s", err)
	}
}
