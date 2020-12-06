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

// Package tools provide some useful utilites used by dgo.
package tools

import (
	"io/ioutil"
	"os"
	"path"
	"strings"
)

// IsDocker - tells we are running from inside docker or other container.
func IsDocker() bool {
	return isDockerFileExists() || isDockerHasCGroup()
}

func InitDockerfile(root, cmd string) error {
	p := path.Join(root, "Dockerfile."+cmd)
	const template = `FROM golang:1.13
COPY dist/APP go/bin/APP
CMD APP`

	source := strings.ReplaceAll(template, "APP", cmd)
	return ioutil.WriteFile(p, []byte(source), os.ModePerm)
}

func isDockerFileExists() bool {
	_, err := os.Stat("/.dockerenv")
	if err != nil {
		return false
	}
	return os.IsExist(err)
}

func isDockerHasCGroup() bool {
	content, err := ioutil.ReadFile("/proc/self/cgroup")
	if err != nil {
		return false
	}
	text := string(content)
	return strings.Contains(text, "docker") || strings.Contains(text, "lxc")
}
