package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"slices"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/topi314/gobin/v2/gobin"
	"github.com/topi314/gobin/v2/internal/ezhttp"
)

func NewShareCmd(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:     "share",
		GroupID: "actions",
		Short:   "Shares a document",
		Example: `gobin share -p write -p delete -p share jis74978

Will create a new share the document jis74978 with the permissions write, delete and share`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: documentCompletion,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := viper.BindPFlag("server", cmd.Flags().Lookup("server")); err != nil {
				return err
			}
			if err := viper.BindPFlag("token", cmd.Flags().Lookup("token")); err != nil {
				return err
			}
			return viper.BindPFlag("permissions", cmd.Flags().Lookup("permissions"))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			documentID := args[0]
			server := viper.GetString("server")
			token := viper.GetString("token")
			permissions := viper.GetStringSlice("permissions")

			if len(permissions) == 0 {
				cmd.Printf("Link: %s/%s\n", server, documentID)
				return nil
			}

			if token == "" {
				token = viper.GetString("tokens_" + documentID)
			}
			if token == "" {
				return fmt.Errorf("no token found or provided for document: %s", documentID)
			}

			perms := make([]string, len(permissions))
			for i, perm := range permissions {
				if !slices.Contains(gobin.AllStringPermissions, perm) {
					return fmt.Errorf("invalid permission: %s", perm)
				}
				perms[i] = perm
			}

			shareRq := gobin.ShareRequest{
				Permissions: perms,
			}

			buff := new(bytes.Buffer)
			if err := json.NewEncoder(buff).Encode(shareRq); err != nil {
				return fmt.Errorf("failed to encode share request: %w", err)
			}

			rs, err := ezhttp.PostToken("/documents/"+documentID+"/share", token, buff)
			if err != nil {
				return fmt.Errorf("failed to create share token: %w", err)
			}

			var shareRs gobin.ShareResponse
			if err = ezhttp.ProcessBody("create share token", rs, &shareRs); err != nil {
				return err
			}

			cmd.Printf("Link: %s/%s?token=%s\n", server, documentID, shareRs.Token)
			return nil
		},
	}

	parent.AddCommand(cmd)

	cmd.Flags().StringP("server", "s", "", "Gobin server address")
	cmd.Flags().StringP("token", "t", "", "The token for the document")
	cmd.Flags().StringSliceP("permissions", "p", nil, "The permissions for the document")

	if err := cmd.RegisterFlagCompletionFunc("permissions", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return gobin.AllStringPermissions, cobra.ShellCompDirectiveNoFileComp
	}); err != nil {
		log.Printf("failed to register permissions flag completion func: %s", err)
	}
}
