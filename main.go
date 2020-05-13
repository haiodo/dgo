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

// Package main - define a dgo helper build application.
package main

import (
	"github.com/haiodo/dgo/cmd/dgo"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.Infof("dgo version %v", 1)
	err := dgo.Execute()
	if err != nil {
		logrus.Fatalf("error executing root command: %v", err)
	}
}
