package dgo

import (
	"github.com/haiodo/dgo/cmd/dgo/tools"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"path"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Creates default Dockerfile for dgo",
	Long:  "Creates default Dockerfile for dgo",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, cgoEnv := tools.RetrieveGoEnv(cmdArguments.cgoEnabled, cmdArguments.goos, cmdArguments.goarch)
		curDir, err := os.Getwd()
		if err != nil {
			logrus.Errorf("Failed to receive current dir %v", err)
			return err
		}
		if len(args) == 0 {
			args = tools.FindMainPackages(cmd.Context(), curDir, cgoEnv)
		}

		for _, arg := range args {
			rootDir := path.Clean(arg)
			_, cmdName := path.Split(rootDir)
			if err := tools.InitDockerfile(rootDir, cmdName); err != nil {
				return err
			}
		}
		return nil
	},
}
