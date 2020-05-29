// SPDX-FileCopyrightText: 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package oic

import (
	"os"
	"os/exec"
	"strings"

	"github.com/lirios/ostree-image-creator/internal/logger"
)

// RunCommand logs the arguments and executes the command cmd.
func RunCommand(cmd *exec.Cmd) error {
	logger.Debugf("+ %s", strings.Join(cmd.Args, " "))

	// Print output
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// RunCommandWithOutput logs the arguments and executes the command cmd
// and saves the output into outFile.
func RunCommandWithOutput(cmd *exec.Cmd, outFile *os.File) error {
	logger.Debugf("+ %s", strings.Join(cmd.Args, " "))

	// Print output
	cmd.Stdout = outFile
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
