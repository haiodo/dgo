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
	"fmt"
	"github.com/haiodo/dgo/cmd/dgo/tools"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"path"
	"sync"
)

type BuildCmdArguments struct {
	outputFolder string
	compileTests bool

	goarch     string
	goos       string
	cgoEnabled bool
	docker     bool
}

var cmdArguments = &BuildCmdArguments{}

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().StringVarP(&cmdArguments.outputFolder,
		"output", "o", "./dist", "Output folder")

	buildCmd.Flags().BoolVarP(&cmdArguments.compileTests,
		"tests", "t", true, "Compile individual test packages")

	buildCmd.Flags().BoolVarP(&cmdArguments.docker,
		"docker", "", true, "If enabled, will do docker build . --build-arg DGO_SKIP_BUILD=true after local build will be done")

	buildCmd.Flags().BoolVarP(&cmdArguments.cgoEnabled,
		"cgo", "", false, "If disabled will pass CGO_ENABLED=0 env variable to go compiler")

	buildCmd.Flags().StringVarP(&cmdArguments.goos,
		"goos", "", "linux", "If passed will pass GOOS=${value} env variable")

	buildCmd.Flags().StringVarP(&cmdArguments.goarch,
		"goarch", "", "amd64", "If passed will pass GOARCH=${value} env variable")
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Perform a build of passed applications and tests CGO_ENABLED=0 GOOS=linux GOARCH=amd64",
	Long:  "Perform a build of passed application and all tests related to it with CGO_ENABLED=0 GOOS=linux GOARCH=amd64",
	RunE: func(cmd *cobra.Command, args []string) error {
		logrus.Infof("dgo.build target...")
		err := PerformBuild(cmd, args, cmdArguments)
		if err != nil {
			logrus.Infof("Build complete")
		}
		return err
	},
}

func PerformBuild(cmd *cobra.Command, args []string, cmdArguments *BuildCmdArguments) error {
	if os.Getenv(SkipBuildEnv) == "true" {
		logrus.Infof("Build is complete on host. Success.")
		return nil
	}

	env, cgoEnv := tools.RetrieveGoEnv(cmdArguments.cgoEnabled, cmdArguments.goos, cmdArguments.goarch)

	curDir, err := os.Getwd()
	if err != nil {
		logrus.Errorf("Failed to receive current dir %v", err)
		return err
	}

	if len(args) == 0 {
		args = tools.FindMainPackages(cmd.Context(), curDir, cgoEnv)
	}

	var wg sync.WaitGroup
	var pkgError error
	for _, root := range args {
		rootDir := path.Clean(root)
		_, cmdName := path.Split(path.Clean(rootDir))

		wg.Add(1)
		go func() {
			defer wg.Done()
			logrus.Infof("Building: %v at %v", cmdName, rootDir)
			buildCmd := []string{"go", "build", "-o", path.Join(cmdArguments.outputFolder, cmdName), rootDir}
			if err := tools.Exec(cmd.Context(), curDir, buildCmd, env); err != nil {
				logrus.Errorf("Error build: %v %v", buildCmd, err)
				pkgError = err
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			testPackages, err := tools.FindTests(cmd.Context(), rootDir, cgoEnv)
			if err != nil {
				pkgError = err
				return
			}
			for k, p := range testPackages {
				if len(p.Tests) > 0 {
					pp := p
					logrus.Infof("Found tests: %v for package: %v", pp.Tests, k)
					if cmdArguments.compileTests {
						wg.Add(1)
						go func() {
							defer wg.Done()
							testPath := path.Join(rootDir, pp.RelPath)
							buildCmd := []string{"go", "test", "-c", "-o", path.Join(cmdArguments.outputFolder, pp.OutName), testPath}
							if err := tools.Exec(cmd.Context(), curDir, buildCmd, env); err != nil {
								logrus.Errorf("Error build: %v %v", buildCmd, err)
								pkgError = err
								return
							}
							logrus.Infof("Compile of %v from %v complete", pp.OutName, testPath)
						}()
					}
				}
			}
		}()
	}
	wg.Wait()
	if pkgError != nil {
		logrus.Errorf("Build failed %v", pkgError)
		return pkgError
	}

	if cmdArguments.docker && !tools.IsDocker() {
		logrus.Infof("Building docker container")
		err := tools.Exec(cmd.Context(), curDir, []string{"docker", "build", "--build-arg", fmt.Sprintf("%s=true", SkipBuildEnv), "."}, env)
		if err != nil {
			logrus.Errorf("Failed to build docker container %v", err)
			return err
		}
	}
	return nil
}
