// SPDX-FileCopyrightText: 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lirios/ostree-image-creator/internal/oic"
	"github.com/lirios/ostree-image-creator/internal/ostree"
)

// fileEntry represents an entry of the files list
type fileEntry struct {
	Glob string      `yaml:"glob"`
	Mode os.FileMode `yaml:"mode,omitempty"`
}

// OstreeCheckoutAction represents an action called "ostree-checkout"
type OstreeCheckoutAction struct {
	BaseAction `yaml:",inline"`
	Repository string      `yaml:"repository"`
	Branch     string      `yaml:"branch"`
	From       string      `yaml:"from"`
	To         string      `yaml:"to"`
	Files      []fileEntry `yaml:"files"`
}

// Validate checks if the action is configured correctly
func (a *OstreeCheckoutAction) Validate(context *oic.Context) error {
	if a.Repository == "" {
		return fmt.Errorf("property \"repository\" is mandatory for the \"%s\" action", a)
	}
	a.Repository = filepath.Join(context.WorkspaceDir, a.Repository)

	if a.Branch == "" {
		return fmt.Errorf("property \"branch\" is mandatory for the \"%s\" action", a)
	}

	if a.From == "" {
		return fmt.Errorf("property \"from\" is mandatory for the \"%s\" action", a)
	}

	if a.To == "" {
		return fmt.Errorf("property \"to\" is mandatory for the \"%s\" action", a)
	}
	a.To = filepath.Join(context.WorkspaceDir, a.To)
	if !strings.HasPrefix(a.To, context.WorkspaceDir) {
		return fmt.Errorf("property \"to\" must contain a path relative to the working directory")
	}

	return nil
}

// Run runs the action
func (a *OstreeCheckoutAction) Run(context *oic.Context) error {
	if err := os.MkdirAll(a.To, 0755); err != nil {
		return err
	}

	repo, err := ostree.OpenRepo(a.Repository)
	if err != nil {
		return err
	}
	defer repo.Close()

	commit, err := repo.ResolveRev(a.Branch)
	if err != nil {
		return err
	}

	err = repo.Walk(commit, a.From, func(path string) error {
		// Does it match the glob?
		for _, file := range a.Files {
			matched, err := filepath.Match(file.Glob, filepath.Base(path))
			if err != nil {
				return err
			}

			// Checkout file if matched
			if matched {
				if err := repo.Checkout(commit, path, a.To); err != nil {
					return err
				}

				if file.Mode > 0 {
					os.Chmod(filepath.Join(a.To, filepath.Base(path)), file.Mode)
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
