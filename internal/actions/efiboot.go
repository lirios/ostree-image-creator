// SPDX-FileCopyrightText: 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/lirios/ostree-image-creator/internal/oic"
	"github.com/lirios/ostree-image-creator/internal/ostree"
)

// EfiBootAction represents an action called "download"
type EfiBootAction struct {
	BaseAction `yaml:",inline"`
	Repository string `yaml:"repository"`
	Branch     string `yaml:"branch"`
	Path       string `yaml:"path"`
}

// Validate checks if the action is configured correctly
func (a *EfiBootAction) Validate(context *oic.Context) error {
	if a.Repository == "" {
		return fmt.Errorf("property \"repository\" is mandatory for the \"%s\" action", a)
	}
	a.Repository = filepath.Join(context.WorkspaceDir, a.Repository)

	if a.Branch == "" {
		return fmt.Errorf("property \"branch\" is mandatory for the \"%s\" action", a)
	}

	if a.Path == "" {
		return fmt.Errorf("property \"path\" is mandatory for the \"%s\" action", a)
	}
	a.Path = filepath.Join(context.WorkspaceDir, a.Path)

	return nil
}

// Run runs the action
func (a *EfiBootAction) Run(context *oic.Context) error {
	if err := os.MkdirAll(filepath.Dir(a.Path), 0755); err != nil {
		return err
	}

	// Temporary directory
	tmpDir := filepath.Join(context.ScrapDir, "efiboot")

	// Open repository
	repo, err := ostree.OpenRepo(a.Repository)
	if err != nil {
		return err
	}
	defer repo.Close()

	// Resolve branch
	rev, err := repo.ResolveRev(a.Branch)
	if err != nil {
		return err
	}

	// Extract files
	if err := repo.Checkout(rev, "/usr/lib/ostree-boot/efi/EFI", tmpDir); err != nil {
		return err
	}

	// Estimate size
	size, err := estimateDirectorySize(tmpDir, 25)
	if err != nil {
		os.RemoveAll(tmpDir)
		return err
	}

	// Create image of the right size
	file, err := os.Create(a.Path)
	if err != nil {
		os.RemoveAll(tmpDir)
		return err
	}
	if err := file.Truncate(size); err != nil {
		file.Close()
		os.RemoveAll(tmpDir)
		return err
	}
	file.Close()

	// Make the file system
	if err := executeCommand(context, "mkfs.msdos", a.Path); err != nil {
		os.RemoveAll(tmpDir)
		os.Remove(a.Path)
		return err
	}

	// Mount
	mountPoint, err := ioutil.TempDir(context.ScrapDir, "mountpoint-")
	if err != nil {
		os.RemoveAll(tmpDir)
		os.Remove(a.Path)
		return err
	}
	if err := executeCommand(context, "mount", "-n", a.Path, mountPoint, "-oloop"); err != nil {
		os.RemoveAll(tmpDir)
		os.Remove(a.Path)
		return err
	}

	// Create destination directory
	destPath := filepath.Join(mountPoint, "EFI")
	if err := os.MkdirAll(destPath, 0755); err != nil {
		os.RemoveAll(tmpDir)
		os.Remove(a.Path)
		return err
	}

	// Copy files
	cmd := exec.Command("cp", "-R", "-L", "--preserve=timestamps", ".", destPath)
	cmd.Dir = tmpDir
	if err := oic.RunCommand(cmd); err != nil {
		return err
	}

	// Cleanup
	executeCommand(context, "umount", mountPoint)
	os.RemoveAll(mountPoint)
	os.RemoveAll(tmpDir)

	return nil
}
