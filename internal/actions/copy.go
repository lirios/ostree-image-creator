// SPDX-FileCopyrightText: 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"syscall"

	"github.com/lirios/ostree-image-creator/internal/oic"
)

// CopyAction represents an action called "download"
type CopyAction struct {
	BaseAction `yaml:",inline"`
	From       string   `yaml:"from"`
	To         string   `yaml:"to"`
	Exclude    []string `yaml:"exclude,omitempty"`
}

// Validate checks if the action is configured correctly
func (a *CopyAction) Validate(context *oic.Context) error {
	if a.From == "" {
		return fmt.Errorf("property \"from\" is mandatory for the \"%s\" action", a)
	}
	a.From = filepath.Join(context.ManifestDir, a.From)

	if a.To == "" {
		return fmt.Errorf("property \"to\" is mandatory for the \"%s\" action", a)
	}
	a.To = filepath.Join(context.WorkspaceDir, a.To)

	return nil
}

// Run runs the action
func (a *CopyAction) Run(context *oic.Context) error {
	if err := os.MkdirAll(a.To, 0755); err != nil {
		return err
	}

	return copyTree(a.From, a.To, a.Exclude)
}

// Cleanup cleans up after the action
func (a *CopyAction) Cleanup(context *oic.Context) error {
	return nil
}

// copyFile copies the source file to the dest destination.
// The copy is as atomic as possible: the file is copied into a temporary file that
// is later renamed.  Permissions and ownership are preserved during the copy.
func copyFile(source, dest string) error {
	// Get source file information
	fileInfo, err := os.Stat(source)
	if err != nil {
		return err
	}

	// Open source file
	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Create a temporary destination file
	destFile, err := ioutil.TempFile(filepath.Dir(dest), "")
	if err != nil {
		return err
	}

	// Copy data from source to temporary destination
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		destFile.Close()
		os.Remove(destFile.Name())
		return err
	}

	// Close the temporary destination
	if err = destFile.Close(); err != nil {
		os.Remove(destFile.Name())
		return err
	}

	// Copy permissions to the temporary destination
	if err = os.Chmod(destFile.Name(), fileInfo.Mode()); err != nil {
		os.Remove(destFile.Name())
		return err
	}

	// Attempt to copy ownership to the temporary destination
	if stat, ok := fileInfo.Sys().(*syscall.Stat_t); ok {
		if err = os.Chown(destFile.Name(), int(stat.Uid), int(stat.Gid)); err != nil {
			return err
		}
	}

	// Now that data and attributes are copied, we can rename to the desired name
	if err = os.Rename(destFile.Name(), dest); err != nil {
		os.Remove(destFile.Name())
		return err
	}

	return nil
}

// copyTree walks the source tree and copies all files to dest but those in the exclude list.
func copyTree(source, dest string, exclude []string) error {
	f := func(sourcePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Base name
		baseName, err := filepath.Rel(source, sourcePath)
		if err != nil {
			return err
		}

		// Exlude the file if it's in the list
		for _, excludedFileName := range exclude {
			if baseName == excludedFileName {
				return nil
			}
		}

		// Destination path
		destPath := path.Join(dest, baseName)

		switch info.Mode() & os.ModeType {
		case 0:
			// Copy a file
			if err = copyFile(sourcePath, destPath); err != nil {
				return err
			}
		case os.ModeDir:
			// Create directory with the same permission
			os.Mkdir(destPath, info.Mode())
		case os.ModeSymlink:
			// Copy symbolic link
			link, err := os.Readlink(sourcePath)
			if err != nil {
				return err
			}
			os.Symlink(link, destPath)
		default:
			return fmt.Errorf("cannot copy %s: unsupported file type", sourcePath)
		}

		return nil
	}
	return filepath.Walk(source, f)
}
