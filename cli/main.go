package main

import (
	"time"

	"github.com/topi314/gobin/v2/cmd"
	"github.com/topi314/gobin/v2/gobin"
)

// These variables are set via the -ldflags option in go build
var (
	Version   = "unknown"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	buildTime, _ := time.Parse(time.RFC3339, BuildTime)

	rootCmd := cmd.NewRootCmd()
	cmd.NewGetCmd(rootCmd)
	cmd.NewPostCmd(rootCmd)
	cmd.NewRmCmd(rootCmd)
	cmd.NewImportCmd(rootCmd)
	cmd.NewShareCmd(rootCmd)
	cmd.NewVersionCmd(rootCmd, gobin.FormatBuildVersion(Version, Commit, buildTime))
	cmd.NewEnvCmd(rootCmd)
	cmd.NewCompletionCmd(rootCmd)
	cmd.Execute(rootCmd)
}
