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
	"github.com/haiodo/dgo/cmd/dgo/spire"
	"github.com/haiodo/dgo/cmd/dgo/tools"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
)

const (
	SpireInitDone  = "DGO:Spire Initialization Done"
	DebugEnv       = "DGO_TEST_DEBUG"
	TestPackageEnv = "DGO_TEST_PACKAGE"
	SkipBuildEnv   = "DGO_SKIP_BUILD"
)

var testArguments = struct {
	outputFolder string

	spire      bool
	cgoEnabled bool

	debugTests  bool
	testPackage string
}{}

func init() {
	cmd := testCmd
	rootCmd.AddCommand(cmd)

	testCmd.Flags().StringVarP(&testArguments.outputFolder,
		"output", "o", "./dist", "Output folder (default ./dist)")

	testCmd.Flags().BoolVarP(&testArguments.cgoEnabled,
		"cgo", "", false, "If disabled will pass CGO_ENABLED=0 env variable to go compiler (default disabled)")

	testCmd.Flags().BoolVarP(&testArguments.spire,
		"spire", "s", true, "If enabled will run spire (default enabled)")

	testCmd.Flags().BoolVarP(&testArguments.debugTests,
		"debug", "d", false, "If enabled will start debug for every test we run with dlv")

	testCmd.Flags().StringVarP(&testArguments.testPackage,
		"test", "t", "", "Run tests only for specified package")
}

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Perform a running of all tests",
	Long:  `Perform a running of all tests found in /bin/*.test`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logrus.Infof("dgo.test target...")
		isDocker := tools.IsDocker()

		if isDocker {
			return testOnDocker(cmd, args)
		}
		return testOnHost(cmd, args)
	},
}

func testOnHost(cmd *cobra.Command, args []string) error {
	curDir, err := os.Getwd()
	if err != nil {
		logrus.Errorf("Failed to receive current dir %v", err)
		return err
	}

	_, cgoEnv := tools.RetrieveGoEnv(cmdArguments.cgoEnabled, cmdArguments.goos, cmdArguments.goarch)

	if len(args) == 0 {
		//We look for packages only in docker env, else we run only /bin/*.test applications.
		args = tools.FindMainPackages(cmd.Context(), curDir, cgoEnv)
	}

	// we need to perform local build before we will start testing in docker container.
	if err = PerformBuild(cmd, args, &BuildCmdArguments{
		cgoEnabled:   testArguments.cgoEnabled,
		goos:         cmdArguments.goos,
		goarch:       cmdArguments.goarch,
		docker:       false,
		outputFolder: testArguments.outputFolder,
		compileTests: true,
	}); err != nil {
		logrus.Errorf("Failed to build %v", err)
		return err
	}

	containerId := ""
	containerId, err = buildTarget(cmd.Context(), curDir, "test", cgoEnv)
	if err != nil {
		return err
	}

	// Remove running containers
	var containers []string
	containers, err = tools.ExecRead(cmd.Context(), curDir, []string{"docker", "ps", "--filter", "label=dgo.test"}, nil, false)
	if err != nil {
		return err
	}
	for _, c := range containers {
		if !strings.HasPrefix(c, "CONTAINER") {
			killId := strings.Split(c, " ")[0]
			if len(killId) > 0 {
				logrus.Infof("Killing container %s", killId)
				if err := tools.Exec(cmd.Context(), curDir, []string{"docker", "kill", killId}, nil); err != nil {
					return err
				}
			}
		}
	}

	runCmd := []string{"docker", "run"}

	if testArguments.testPackage != "" {
		runCmd = append(runCmd, "-e", fmt.Sprintf("%s=%s", TestPackageEnv, testArguments.testPackage))
	}

	if testArguments.debugTests {
		runCmd = append(runCmd, "-e", DebugEnv+"=:40000", "-p", "40000:40000")
	}

	runCmd = append(runCmd, "--label", "dgo.test", "--rm", containerId)

	err = tools.Exec(cmd.Context(), curDir, runCmd, nil)
	if err != nil {
		logrus.Errorf("Failed to run docker run %v cause: %v", containerId, err)
		return err
	}
	return nil
}

// DEBUG:
//  docker run -e DLV_LISTEN_NSM=:40000 -p 40000:40000 $(docker build -q . --target test)

func testOnDocker(cmd *cobra.Command, args []string) error {
	// We are inside docker, let's find all test applications.
	// Find applications and tests from its
	curDir, err := os.Getwd()
	if err != nil {
		logrus.Errorf("Failed to receive current dir %v", err)
		return err
	}

	packages := map[string]map[string]*tools.PackageInfo{}
	files, err := ioutil.ReadDir("/bin")
	if err != nil {
		logrus.Fatalf("failed to list /bin cause: %v", err)
	}

	for _, f := range files {
		fName := f.Name()
		const testSuffix = ".test"
		if strings.HasSuffix(fName, testSuffix) {
			// This is probable go test, let's find out a tests inside and extract cmdName.
			cmdName := fName[0 : len(fName)-len(testSuffix)]
			relPath := ""
			// Remove package name
			sepPos := strings.Index(cmdName, "-")
			if sepPos != -1 {
				relPath = fName[sepPos+1 : len(fName)]
				cmdName = fName[0:sepPos]
			}
			pkgRoot, ok := packages[cmdName]
			if !ok {
				pkgRoot = map[string]*tools.PackageInfo{}
				packages[cmdName] = pkgRoot
			}
			pkgInfo := &tools.PackageInfo{
				OutName: f.Name(),
				RelPath: strings.ReplaceAll(relPath, "-", "/"),
			}

			// Check if main application are present, since we need it as SUT
			if _, err := os.Stat(path.Join("/bin", cmdName)); os.IsNotExist(err) {
				logrus.Infof("Test file %v ignored, since no main application found %v: cause: %v", f.Name(), path.Join("/bin", cmdName), err)
				continue
			}

			lines, err := tools.ExecRead(cmd.Context(), curDir, []string{"/bin/" + pkgInfo.OutName, "-test.list", ".*"}, nil, false)
			if err != nil {
				logrus.Errorf("Failed to list test for %v cause: %v", pkgInfo.OutName, err)
			}
			for _, t := range lines {
				t = strings.TrimSpace(t)
				if len(t) > 0 {
					pkgInfo.Tests = append(pkgInfo.Tests, t)
				}
			}
			logrus.Infof("Found tests for %v %v", pkgInfo.OutName, pkgInfo.Tests)

			pkgRoot[relPath] = pkgInfo
		}
	}

	if testArguments.spire {
		// We are inside docker, so spire should be available and we just need to run it.
		// Run spire
		agentID := "spiffe://example.org/myagent"
		spireCtx, err := spire.New("", agentID)
		if err != nil {
			logrus.Errorf("failed to start spire: %v", err)
		}
		err = spireCtx.Start(cmd.Context())
		if err != nil {
			logrus.Fatalf("failed to run spire: %+v", err)
		}
		for cmdName, pkgs := range packages {
			// Add spire entries for every appliction and test application we found.
			if err = spireCtx.AddEntry(agentID, fmt.Sprintf("spiffe://example.org/%s", cmdName), fmt.Sprintf("unix:path:/bin/%s", cmdName)); err != nil {
				logrus.Fatalf("failed to add entry to spire: %+v", err)
			}

			if err = spireCtx.AddEntry(agentID, fmt.Sprintf("spiffe://example.org/dlv"), fmt.Sprintf("unix:path:/bin/dlv")); err != nil {
				logrus.Fatalf("failed to add entry to spire: %+v", err)
			}

			if err = spireCtx.AddEntry(agentID, fmt.Sprintf("spiffe://example.org/any-test"), fmt.Sprintf("unix:uid:0")); err != nil {
				logrus.Fatalf("failed to add entry to spire: %+v", err)
			}

			for _, info := range pkgs {
				if len(info.Tests) > 0 {
					if err = spireCtx.AddEntry(agentID, fmt.Sprintf("spiffe://example.org/%s", info.OutName),
						fmt.Sprintf("unix:path:/bin/%s", info.OutName)); err != nil {
						logrus.Fatalf("failed to add entry to spire: %+v", err)
					}
				}
			}
			logrus.Info(SpireInitDone)
		}
	}

	debugCmd := []string{}

	listenArg := os.Getenv(DebugEnv)
	if listenArg != "" {
		// Do we have dlv?
		dlv, err := exec.LookPath("dlv")
		if err != nil {
			return errors.Wrap(err, "Unable to find dlv in your path")
		}

		// Marshal the new args
		debugCmd = []string{dlv, "--listen=" + listenArg, "--headless=true", "--api-version=2", "exec"}
	}

	testPkg := os.Getenv(TestPackageEnv)
	if len(testPkg) > 0 {
		testArguments.testPackage = testPkg
	}

	// Ok we are ready to run tests
	for cmdName, testApp := range packages {
		logrus.Infof("Running tests for %v", cmdName)
		for _, testPkg := range testApp {
			if len(testPkg.Tests) > 0 {
				if testArguments.testPackage != "" && testArguments.testPackage != testPkg.OutName {
					logrus.Infof("Testing of %s is skipped since package are selected %v", testPkg.OutName, testArguments.testPackage)
					continue
				}
				// Run the test
				testExecName := path.Join("/bin", testPkg.OutName)

				execName := append(debugCmd, testExecName)
				// Run the test
				if err := tools.Exec(cmd.Context(), curDir, execName, nil); err != nil {
					logrus.Fatalf("Error running test Executable: %q err: %q", testExecName, err)
					return err
				}
			}
		}
	}
	return nil
}
