// SPDX-FileCopyrightText: 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lirios/ostree-image-creator/internal/oic"
	"github.com/lirios/ostree-image-creator/internal/ostree"
)

// OstreeMirrorAction represents an action called "ostree-mirror"
type OstreeMirrorAction struct {
	BaseAction `yaml:",inline"`
	Repository string   `yaml:"repository"`
	URL        string   `yaml:"url"`
	OSName     string   `yaml:"osname"`
	Branches   []string `yaml:"branches"`
}

// Validate checks if the action is configured correctly
func (a *OstreeMirrorAction) Validate(context *oic.Context) error {
	if a.Repository == "" {
		return fmt.Errorf("property \"repository\" is mandatory for the \"%s\" action", a)
	}
	a.Repository = filepath.Join(context.WorkspaceDir, a.Repository)

	if a.URL == "" {
		return fmt.Errorf("property \"url\" is mandatory for the \"%s\" action", a)
	}

	if a.OSName == "" {
		return fmt.Errorf("property \"osname\" is mandatory for the \"%s\" action", a)
	}

	if len(a.Branches) == 0 {
		return fmt.Errorf("property \"branches\" is mandatory for the \"%s\" action", a)
	}

	return nil
}

// Run runs the action
func (a *OstreeMirrorAction) Run(context *oic.Context) error {
	if err := os.MkdirAll(a.Repository, 0755); err != nil {
		return err
	}

	repo, err := ostree.OpenRepo(a.Repository)
	if err != nil {
		return err
	}

	if !repo.HasRemote(a.OSName) {
		options := ostree.RemoteOptions{NoGPGVerify: true, NoGPGVerifySummary: true}
		if err := repo.RemoteAdd(a.OSName, a.URL, options); err != nil {
			return err
		}
	}

	return repo.Pull(a.OSName, a.Branches, ostree.PullFlags{Mirror: true})
}
