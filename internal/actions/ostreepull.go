// SPDX-FileCopyrightText: 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"fmt"
	"path/filepath"

	"github.com/lirios/ostree-image-creator/internal/oic"
	"github.com/lirios/ostree-image-creator/internal/ostree"
)

// OstreePullAction represents an action called "ostree-pull"
type OstreePullAction struct {
	BaseAction  `yaml:",inline"`
	Repository  string   `yaml:"repository"`
	URL         string   `yaml:"url"`
	OSName      string   `yaml:"osname"`
	Branches    []string `yaml:"branches"`
	NoGPGVerify bool     `yaml:"no-gpg-verify"`
}

// Validate checks if the action is configured correctly
func (a *OstreePullAction) Validate(context *oic.Context) error {
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
func (a *OstreePullAction) Run(context *oic.Context) error {
	repo, err := ostree.OpenRepo(a.Repository)
	if err != nil {
		return err
	}

	if !repo.HasRemote(a.OSName) {
		options := ostree.RemoteOptions{NoGPGVerify: a.NoGPGVerify, NoGPGVerifySummary: a.NoGPGVerify}
		if err := repo.RemoteAdd(a.OSName, a.URL, options); err != nil {
			return err
		}
	}

	pullOptions := ostree.PullOptions{OverrideRemoteName: a.OSName, Refs: a.Branches}
	return repo.PullWithOptions(a.OSName, pullOptions)
}
