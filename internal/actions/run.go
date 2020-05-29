// SPDX-FileCopyrightText: 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"

	"github.com/lirios/ostree-image-creator/internal/oic"
)

// RunAction represents an action called "run"
type RunAction struct {
	BaseAction `yaml:",inline"`
	WorkDir    string `yaml:"working-directory,omitempty"`
	Shell      string `yaml:"shell"`
	Command    string `yaml:"command"`
	Script     string `yaml:"script"`
}

// Validate checks if the action is configured correctly
func (a *RunAction) Validate(context *oic.Context) error {
	if a.Command != "" && a.Script != "" {
		return fmt.Errorf("cannot specify both \"command\" and \"script\" properties for the \"%s\" action", a)
	}

	if a.Command == "" && a.Script == "" {
		return fmt.Errorf("either \"command\" or \"script\" properties is mandatory for the \"%s\" action", a)
	}

	if a.WorkDir != "" {
		a.WorkDir = filepath.Join(context.WorkspaceDir, a.WorkDir)
	}

	if a.Shell == "" {
		a.Shell = "bash"
	}

	return nil
}

// Run runs the action
func (a *RunAction) Run(context *oic.Context) error {
	if a.Command != "" {
		scriptFile, err := ioutil.TempFile(context.ScrapDir, "script-")
		if err != nil {
			return err
		}
		defer scriptFile.Close()
		if a.Shell == "bash" {
			if _, err := scriptFile.Write([]byte("set -e\n")); err != nil {
				return err
			}
		}
		if _, err := scriptFile.Write([]byte(a.Command)); err != nil {
			return err
		}
		scriptFile.Close()

		cmd := exec.Command("/bin/env", a.Shell, scriptFile.Name())
		if a.WorkDir != "" {
			cmd.Dir = a.WorkDir
		}
		return oic.RunCommand(cmd)
	}

	cmd := exec.Command(a.Script)
	return oic.RunCommand(cmd)
}
