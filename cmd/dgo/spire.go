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
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spiffe/go-spiffe/workload"
	"os"
)

func init() {
	cmd := spireCmd
	rootCmd.AddCommand(cmd)

}

var spireCmd = &cobra.Command{
	Use:   "spire",
	Short: "Running a spire server with default settings",
	Long:  `Running a spire server with default settings`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logrus.Infof("NSM.Spire target...")

		agentID := "spiffe://example.org/myagent"
		spireContext, err := spire.New("", agentID)
		if err != nil {
			logrus.Errorf("Error: %v", err)
			return err
		}

		err = spireContext.Start(cmd.Context())
		if err != nil {
			logrus.Fatalf("failed to run spire: %+v", err)
		}

		var curUserId []string
		curUserId, err = tools.ExecRead(cmd.Context(), "", []string{"id", "-u"}, nil)

		if err = spireContext.AddEntry(agentID, fmt.Sprintf("spiffe://example.org/test"), fmt.Sprintf("unix:uid:%s", curUserId[0])); err != nil {
			logrus.Fatalf("failed to add entry to spire: %+v", err)
		}

		_, _ = os.Stdout.WriteString(fmt.Sprintf("\n\n************\n\nSpire is up and running, please set ENV variable:\n%s=%s\n\n\n*********\n", workload.SocketEnv, os.Getenv(workload.SocketEnv)))
		<-cmd.Context().Done()

		return nil
	},
}
