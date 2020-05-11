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
	"bufio"
	"context"
	"errors"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"os/exec"
	"strings"
)

//wrapper - A simple process wrapper
type wrapper struct {
	Cmd    *exec.Cmd
	cancel context.CancelFunc
	Stdout io.ReadCloser
	Stderr io.ReadCloser
	ctx    context.Context
}

// ExecRead - execute command and return output as result, stderr is ignored.
func ExecRead(ctx context.Context, dir string, args, env []string) ([]string, error) {
	var err error
	if dir == "" {
		dir, err = os.Getwd()
		if err != nil {
			logrus.Errorf("Failed to receive current dir %v", err)
			return nil, err
		}
	}
	var proc *wrapper
	proc, err = execProc(ctx, dir, args, env)
	if err != nil {
		return nil, err
	}
	output := []string{}
	reader := bufio.NewReader(proc.Stdout)
	for {
		s, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		output = append(output, strings.TrimSpace(s))
	}
	err = proc.Cmd.Wait()
	if err != nil {
		return output, err
	}
	return output, nil
}

// Exec - execute shell command
func Exec(ctx context.Context, dir string, args, env []string) error {
	p, err := execProc(ctx, dir, args, env)
	if err != nil {
		return err
	}
	err = p.Cmd.Wait()
	return err
}

// Exec - execute shell command
func Start(ctx context.Context, dir string, args, env []string) (context.Context, error) {
	p, err := execProc(ctx, dir, args, env)
	if err != nil {
		return nil, err
	}
	return p.ctx, err
}

// execProc - execute shell command and return wrapper
func execProc(ctx context.Context, dir string, args, env []string) (*wrapper, error) {
	if len(args) == 0 {
		return nil, errors.New("missing command to run")
	}

	p := &wrapper{

	}
	p.ctx, p.cancel = context.WithCancel(ctx)
	p.Cmd = exec.CommandContext(ctx, args[0], args[1:]...)
	p.Cmd.Dir = dir
	if env != nil {
		p.Cmd.Env = append(os.Environ(), env...)
	}
	var err error
	p.Stdout, err = p.Cmd.StdoutPipe()
	if err != nil {
		return p, err
	}
	p.Stderr, err = p.Cmd.StderrPipe()
	if err != nil {
		return p, err
	}
	err = p.Cmd.Start()
	return p, err
}
