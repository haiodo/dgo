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
// Package spire provides two simple functions:
//   - Start to start a SpireServer/SpireAgent for local testing
//   - AddEntry to add entries into the spire server
package spire

import (
	"context"
	"github.com/haiodo/dgo/cmd/dgo/tools"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/pkg/errors"
)

type SpireContext interface {
	AddEntry(parentID, spiffeID, selector string) error
	Start(ctx context.Context) error
}

type spireContext struct {
	spireRoot       string
	ctx             context.Context
	cancel          context.CancelFunc
	spireSocketPath string
	needClean       bool
	spireServerCtx  context.Context
	agentID         string
	regSocket       string
}

// New - contruct a new spire context
func New(spireRoot string, agentID string) (SpireContext, error) {
	needClean := false
	if spireRoot == "" {
		var err error
		spireRoot, err = ioutil.TempDir("", "spire")
		if err != nil {
			return nil, err
		}
		needClean = true
	}
	_ = os.RemoveAll(spireRoot)
	_ = os.MkdirAll(spireRoot, os.ModePerm)

	return &spireContext{
		spireRoot: spireRoot,
		needClean: needClean,
		agentID:   agentID,
	}, nil
}

// AddEntry - adds an entry to the spire server for parentID, spiffeID, and selector
//            parentID is usually the same as the agentID provided to Start()
func (sc *spireContext) AddEntry(parentID, spiffeID, selector string) error {
	cmdStr := []string{
		"spire-server",
		"entry", "create",
		"-parentID", parentID,
		"-spiffeID", spiffeID,
		"-selector", selector,
		"-registrationUDSPath", sc.regSocket}
	return tools.Exec(sc.ctx, sc.spireRoot, cmdStr, nil)
}

// Start - start a spire-server and spire-agent with the given agentId
func (sc *spireContext) Start(ctx context.Context) error {
	// Setup our context
	sc.ctx, sc.cancel = context.WithCancel(ctx)

	// Write the config files (if not present)
	var err error
	sc.spireSocketPath, sc.regSocket, err = writeDefaultConfigFiles(ctx, sc.spireRoot)

	if err != nil {
		sc.Stop()
		return err
	}
	logrus.Infof("Env variable %s=%s are set", SocketEnv, "unix:"+sc.spireSocketPath)
	if err = os.Setenv(SocketEnv, "unix:"+sc.spireSocketPath); err != nil {
		sc.Stop()
		return err
	}

	// Start the Spire Server
	spireCmd := []string{"spire-server", "run", "-config", path.Join(sc.spireRoot, spireServerConfFileName)}
	sc.spireServerCtx, err = tools.Start(sc.ctx, sc.spireRoot, spireCmd, nil)
	if err != nil {
		sc.Stop()
		return err
	}

	// Healthcheck the Spire Server
	healthOk := false
	err = nil
	for i := 0; i < 10; i++ {
		if err = tools.Exec(sc.ctx, sc.spireRoot, []string{"spire-server", "healthcheck", "-registrationUDSPath", sc.regSocket}, nil); err == nil {
			// All is good, break
			healthOk = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !healthOk {
		err = errors.Wrap(err, "Error starting spire-server")
		sc.Stop()
		return err
	}

	// Get the SpireServers Token
	cmdStr := []string{"spire-server", "token", "generate", "-spiffeID", sc.agentID, "-registrationUDSPath", sc.regSocket}
	var lines []string
	lines, err = tools.ExecRead(sc.ctx, sc.spireRoot, cmdStr, nil, true)
	if err != nil {
		err = errors.Wrap(err, "Error acquiring spire-server token")
		sc.Stop()
		return err
	}
	output := strings.Join(lines, "")
	spireToken := strings.Replace(output, "Token:", "", 1)
	spireToken = strings.TrimSpace(spireToken)

	// Start the Spire Agent
	spireAgentCtx, err := tools.Start(sc.ctx, sc.spireRoot, []string{"spire-agent", "run", "-config", spireAgentConfFilename, "-joinToken", spireToken}, nil)
	if err != nil {
		err = errors.Wrap(err, "Error starting spire-agent")
		sc.Stop()
		return err
	}

	// Healthcheck the Spire Agent
	healthOk = false
	err = nil
	for i := 0; i < 10; i++ {
		if err = tools.Exec(sc.ctx, sc.spireRoot, []string{"spire-agent", "healthcheck", "-socketPath", sc.spireSocketPath}, nil); err == nil {
			// All is good, break
			healthOk = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !healthOk {
		err = errors.Wrap(err, "Error starting spire-server")
		sc.Stop()
		return err
	}

	// Cleanup if either server we spawned dies
	go func() {
		select {
		case <-sc.spireServerCtx.Done():
			logrus.Errorf("spireServer quit unexpectedly")
			sc.Stop()
		case <-spireAgentCtx.Done():
			logrus.Errorf("SpireAgent quit unexpectedly")
			sc.Stop()
		case <-ctx.Done():
		}
		sc.Stop()
	}()
	return nil
}

func (sc *spireContext) Stop() {
	sc.cancel()
	if sc.needClean {
		_ = os.RemoveAll(sc.spireRoot)
	}
}

// writeDefaultConfigFiles - write config files into configRoot and return a spire socket file to use
func writeDefaultConfigFiles(ctx context.Context, spireRoot string) (spireSocketName string, regSocket string, err error) {
	spireSocketName = path.Join(spireRoot, spireEndpointSocket)
	regSocket = path.Join(spireRoot, spireServerRegSock)
	configFiles := map[string]string{
		spireServerConfFileName: genServerConf(spireRoot, spireServerRegSock),
		spireAgentConfFilename:  genSpireConfig(spireRoot, spireEndpointSocket),
	}
	for configName, contents := range configFiles {
		filename := path.Join(spireRoot, configName)
		if _, err = os.Stat(filename); os.IsNotExist(err) {
			logrus.Infof("Configuration file: %q not found, using defaults", filename)
			if err = os.MkdirAll(path.Dir(filename), 0700); err != nil {
				return
			}
			if err = ioutil.WriteFile(filename, []byte(contents), 0700); err != nil {
				return
			}
		}
	}
	return
}
