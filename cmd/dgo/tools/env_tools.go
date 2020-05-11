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
	"fmt"
	"github.com/sirupsen/logrus"
)

// RetrieveGoEnv - return environment strings based on go parameters.
func RetrieveGoEnv(cgoEnabled bool, goos, goarch string) (env []string, cgoEnv []string) {
	logrus.Infof("Process env variables CGO_ENABLED=%v GOOS=%v GOARCH=%v", cgoEnabled, goos, goarch)
	if !cgoEnabled {
		cgoEnv = append(cgoEnv, "CGO_ENABLED=0")
		env = cgoEnv
	}
	env = append(env, fmt.Sprintf("GOOS=%s", goos), fmt.Sprintf("GOARCH=%s", goarch))
	return
}
