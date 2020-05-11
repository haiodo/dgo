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
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"path"
	"regexp"
	"strings"
	"time"
)

// FindMainPackages - find all main packages of current working dir.
func FindMainPackages(ctx context.Context, root string, env []string) []string {
	roots := []string{}
	// Find a list of package roots using
	//List commands do not need to pass environment
	listCmd := []string{"go", "list", "-f", "{{.Name}}:{{.Dir}}", "./..."}
	lines, err := ExecRead(ctx, root, listCmd, env)
	if err != nil {
		logrus.Fatalf("failed to find a list of go package roots: %v", err)
	}
	for _, line := range lines {
		trimLine := strings.TrimSpace(line)
		if len(trimLine) == 0 {
			continue
		}
		if strings.HasPrefix(trimLine, "main:") {
			// we found main package, let's add it as root
			roots = append(roots, trimLine[len(root)+6:len(trimLine)])
		}
	}
	return roots
}

func relativePath(rootDir, pkgName string) string {
	ind := strings.Index(pkgName, rootDir)
	if ind != -1 {
		relPath := pkgName[ind+len(rootDir) : len(pkgName)]
		if strings.HasPrefix(relPath, "/") {
			relPath = relPath[1:len(relPath)]
		}
		return relPath
	}
	return pkgName
}

type PackageInfo struct {
	RelPath string
	Tests   []string
	OutName string
}

type TestEvent struct {
	Time    time.Time // encodes as an RFC3339-format string
	Action  string
	Package string
	Test    string
	Elapsed float64 // seconds
	Output  string
}

var alphaReg, _ = regexp.Compile("[^A-Za-z0-9]+")

func FindTests(ctx context.Context, rootDir string, env []string) (map[string]*PackageInfo, error) {
	logrus.Infof("Find Tests in %v", rootDir)
	testPackages := map[string]*PackageInfo{}
	// Find all Tests
	// List commands do not need to pass environment
	listCmd := []string{"go", "test", "--list", ".*", "-json", "./" + rootDir}
	lines, err := ExecRead(ctx, rootDir, listCmd, env)
	if err != nil {
		logrus.Errorf("Failed to list Tests %v %v", listCmd, err)
		return nil, err
	}

	_, cmdName := path.Split(path.Clean(rootDir))

	for _, line := range lines {
		trimLine := strings.TrimSpace(line)
		if len(trimLine) == 0 {
			continue
		}
		event := TestEvent{}
		err := json.Unmarshal([]byte(trimLine), &event)
		if err != nil {
			logrus.Errorf("Failed to parse line: %v", err)
			continue
		}
		pkgInfo, ok := testPackages[event.Package]
		if !ok {
			relPath := relativePath(rootDir, event.Package)
			outName := fmt.Sprintf("%s-%s.test", cmdName, alphaReg.ReplaceAllString(relPath, "-"))
			if len(relPath) == 0 {
				outName = fmt.Sprintf("%s.test", cmdName)
			}
			pkgInfo = &PackageInfo{
				RelPath: relPath,
				OutName: outName,
				Tests:   []string{},
			}
			testPackages[event.Package] = pkgInfo
		}

		switch event.Action {
		case "output":
			for _, k := range strings.Split(strings.TrimSpace(event.Output), "\n") {
				if strings.HasPrefix(k, "Test") {
					pkgInfo.Tests = append(pkgInfo.Tests, k)
				}
			}
		case "skip":
			pkgInfo.Tests = []string{}
		}
	}
	return testPackages, nil
}
