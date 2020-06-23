// SPDX-FileCopyrightText: 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lirios/ostree-image-creator/internal/logger"
	"github.com/lirios/ostree-image-creator/internal/oic"
)

// OstreeDeployAction represents an action called "ostree-deploy"
type OstreeDeployAction struct {
	BaseAction  `yaml:",inline"`
	Path        string `yaml:"path"`
	URL         string `yaml:"url"`
	OSName      string `yaml:"osname"`
	Branch      string `yaml:"branch"`
	NoGPGVerify bool   `yaml:"no-gpg-verify"`
}

// Validate checks if the action is configured correctly
func (a *OstreeDeployAction) Validate(context *oic.Context) error {
	if a.Path == "" {
		return fmt.Errorf("property \"path\" is mandatory for the \"%s\" action", a)
	}
	a.Path = filepath.Join(context.WorkspaceDir, a.Path)

	if a.URL == "" {
		return fmt.Errorf("property \"url\" is mandatory for the \"%s\" action", a)
	}

	if a.OSName == "" {
		return fmt.Errorf("property \"osname\" is mandatory for the \"%s\" action", a)
	}

	if a.Branch == "" {
		return fmt.Errorf("property \"branch\" is mandatory for the \"%s\" action", a)
	}

	return nil
}

// Run runs the action
func (a *OstreeDeployAction) Run(context *oic.Context) error {
	deployRootDir := filepath.Join(a.Path, "ostree", "deploy")
	if err := os.MkdirAll(deployRootDir, 0755); err != nil {
		return err
	}

	repoDir := filepath.Join(a.Path, "ostree", "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		return err
	}

	// FIXME: With libostree the pull fails with "Writing content object: fsetxattr: Invalid argument",
	// but this doesn't happen launching the ostree binary
	/*
		repo, err := ostree.CreateRepo(repoDir, ostree.RepoModeBare)
		if err != nil {
			return err
		}
		defer repo.Close()

		options := ostree.RemoteOptions{NoGPGVerify: a.NoGPGVerify, NoGPGVerifySummary: a.NoGPGVerify}
		if err := repo.RemoteAdd(a.OSName, a.URL, options); err != nil {
			return err
		}

		pullOptions := ostree.PullOptions{OverrideRemoteName: "origin", Refs: []string{a.Branch}}
		if err := repo.PullWithOptions(a.OSName, pullOptions); err != nil {
			return err
		}
	*/
	if err := executeCommand(context, "ostree", fmt.Sprintf("--repo=%s", repoDir), "init", "--mode=bare"); err != nil {
		return err
	}
	if err := executeCommand(context, "ostree", fmt.Sprintf("--repo=%s", repoDir), "remote", "add", "--no-gpg-verify", "--if-not-exists", a.OSName, a.URL); err != nil {
		return err
	}
	if err := executeCommand(context, "ostree", fmt.Sprintf("--repo=%s", repoDir), "pull", a.OSName, a.Branch); err != nil {
		return err
	}

	if err := executeCommand(context, "ostree", "admin", "os-init", a.OSName, fmt.Sprintf("--sysroot=%s", a.Path)); err != nil {
		return err
	}
	if err := executeCommand(context, "ostree", "admin", "deploy", a.Branch, fmt.Sprintf("--sysroot=%s", a.Path), fmt.Sprintf("--os=%s", a.OSName)); err != nil {
		return err
	}

	// Remove immutable attribute from deployment
	logger.Action("Remove immutable attribute from deployment...")
	deployDir := filepath.Join(deployRootDir, a.OSName, "deploy")
	err := filepath.Walk(deployDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			// Ignore errors because the file system might not support it making the command fail
			// TODO: Restore error checking and execute the command only when it's supported
			executeCommand(context, "chattr", "-i", path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
