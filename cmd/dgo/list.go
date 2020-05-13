// Copyright (c) 2020 Andrey Sobolev.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dgo

import (
	"context"
	"fmt"
	"github.com/haiodo/dgo/cmd/dgo/tools"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var listArguments = struct {
	spire       bool
	cgo_enabled bool
}{}

func init() {
	cmd := listCmd
	rootCmd.AddCommand(cmd)

	listCmd.Flags().BoolVarP(&testArguments.cgoEnabled,
		"cgo", "", false, "If disabled will pass CGO_ENABLED=0 env variable to go compiler")

	listCmd.Flags().BoolVarP(&testArguments.spire,
		"spire", "s", true, "If enabled will run spire")
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Perform a list of available tests",
	Long:  `Perform a list of available tests`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logrus.Infof("dgo.list target...")
		isDocker := tools.IsDocker()

		curDir, err := os.Getwd()
		if err != nil {
			logrus.Errorf("Failed to receive current dir %v", err)
			return err
		}

		_, cgoEnv := tools.RetrieveGoEnv(cmdArguments.cgoEnabled, cmdArguments.goos, cmdArguments.goarch)

		if len(args) == 0 && !isDocker {
			//We look for packages only in docker env, else we run only /bin/*.test applications.
			args = tools.FindMainPackages(cmd.Context(), curDir, cgoEnv)
		}

		// Final All test packages
		packages := map[string]map[string]*tools.PackageInfo{}
		for _, rootDir := range args {
			sourceRoot := rootDir
			rootDir, err = filepath.Abs(rootDir)
			if err != nil {
				logrus.Errorf("Failed to make absolute path for %v", sourceRoot)
			}
			// We in state to run tests,
			var pkgs map[string]*tools.PackageInfo
			pkgs, err = tools.FindTests(cmd.Context(), rootDir, cgoEnv)
			if err != nil {
				logrus.Errorf("failed to find tests %v", err)
			}
			// Add spire entries for every appliction and test application we found.
			_, cmdName := path.Split(path.Clean(rootDir))
			packages[cmdName] = pkgs
		}
		for _, testApp := range packages {
			for _, testPkg := range testApp {
				if len(testPkg.Tests) > 0 {
					// Print test info
					testExecName := path.Join("/bin", testPkg.OutName)
					logrus.Infof("Test binary: %v tests: %v", testExecName, testPkg.Tests)
				}
			}
		}
		return nil
	},
}

func buildTarget(ctx context.Context, curDir, target string, env []string) (containerId string, err error) {
	logrus.Infof("Build target %v with docker...", target)

	var output []string
	output, err = tools.ExecRead(ctx, curDir, []string{"docker", "build", "--build-arg", fmt.Sprintf("%s=true", SkipBuildEnv), ".", "--target", target}, env, true)

	if err != nil {
		return "", err
	}
	lastLine := output[len(output)-1]
	prefix := "Successfully built "
	if strings.HasPrefix(lastLine, prefix) {
		containerId = strings.TrimSpace(lastLine[len(prefix):len(lastLine)])
	} else {
		err = errors.Errorf("Failed to parse container id %v", lastLine)
		return
	}
	return
}
