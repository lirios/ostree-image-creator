// SPDX-FileCopyrightText: 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/lirios/ostree-image-creator/internal/oic"
)

// TemplateAction represents an action called "download"
type TemplateAction struct {
	BaseAction `yaml:",inline"`
	Template   string                 `yaml:"template"`
	Path       string                 `yaml:"path"`
	Variables  map[string]interface{} `yaml:"variables"`
}

// Validate checks if the action is configured correctly
func (a *TemplateAction) Validate(context *oic.Context) error {
	if a.Template == "" {
		return fmt.Errorf("property \"template\" is mandatory for the \"%s\" action", a)
	}
	a.Template = filepath.Join(context.ManifestDir, a.Template)

	if a.Path == "" {
		return fmt.Errorf("property \"path\" is mandatory for the \"%s\" action", a)
	}
	a.Path = filepath.Join(context.WorkspaceDir, a.Path)

	// Add manifest variables
	a.Variables["manifest"] = context.TemplateVars

	return nil
}

// Run runs the action
func (a *TemplateAction) Run(context *oic.Context) error {
	if err := os.MkdirAll(filepath.Dir(a.Path), 0755); err != nil {
		return err
	}

	funcs := template.FuncMap{
		"StringsJoin": strings.Join,
	}

	t, err := template.New(path.Base(a.Template)).Funcs(funcs).ParseFiles(a.Template)
	if err != nil {
		return err
	}

	data := &bytes.Buffer{}
	if err := t.Execute(data, a.Variables); err != nil {
		return err
	}

	file, err := os.Create(a.Path)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(data.Bytes()); err != nil {
		return err
	}

	return nil
}
