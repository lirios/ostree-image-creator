// SPDX-FileCopyrightText: 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package oic

// Context represents the context of the application
type Context struct {
	Architecture string
	ManifestDir  string
	OutputDir    string
	WorkspaceDir string
	ScrapDir     string
	DownloadDir  string
	Keep         bool
	Force        bool
	Verbose      bool
	TemplateVars map[string]interface{}
}
