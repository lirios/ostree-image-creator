// SPDX-FileCopyrightText: 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/lirios/ostree-image-creator/internal/oic"
)

// executeCommand runs a command.
func executeCommand(context *oic.Context, args ...string) error {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = context.WorkspaceDir
	return oic.RunCommand(cmd)
}

// estimateDirectorySize estimates the size of the path directory
// and adds pecent percentage to it.
func estimateDirectorySize(path string, percent uint64) (int64, error) {
	var totalSize int64

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		totalSize += info.Size()
		return nil
	})
	if err != nil {
		return 0, err
	}

	addPercent := (100.0 + float32(percent)) / 100.0
	totalSize = 1 + int64((float32(totalSize) * addPercent))
	return totalSize, nil
}
