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
	"github.com/haiodo/dgo/cmd/dgo/tools"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)

func init() {
	cmd := doCmd
	rootCmd.AddCommand(cmd)
}

var doCmd = &cobra.Command{
	Use:   "do",
	Short: "Perform a do with something if build is allowed",
	Long:  `Perform a do with something if build is allowed`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logrus.Infof("dgo.do target...")
		if os.Getenv(SkipBuildEnv) == "true" {
			logrus.Infof("Do %v is complete on host. Success.", args)
			return nil
		}
		return tools.Exec(cmd.Context(), "", args, nil)
	},
}
