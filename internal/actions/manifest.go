// SPDX-FileCopyrightText: 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
// SPDX-FileCopyrightText: 2017 Collabora Ltd.
//
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"path"
	"strings"

	"gopkg.in/yaml.v2"
)

// Code slightly based on https://github.com/go-debos/debos
// Copyright (C) 2017 Collabora Ltd.
// Originally licensed under the terms of the Apache License 2.0

// ManifestAction extends the Action interface and can select
// the correct implementation of each action at unmarshall time
type ManifestAction struct {
	Action
}

// UnmarshalYAML unmarshalls actions
func (a *ManifestAction) UnmarshalYAML(unmarshall func(interface{}) error) error {
	var action BaseAction

	if err := unmarshall(&action); err != nil {
		return err
	}

	switch action.Action {
	case "copy":
		a.Action = &CopyAction{}
	case "download":
		a.Action = &DownloadAction{}
	case "efiboot":
		a.Action = &EfiBootAction{}
	case "mkiso":
		a.Action = &MkisoAction{}
	case "ostree-checkout":
		a.Action = &OstreeCheckoutAction{}
	case "ostree-deploy":
		a.Action = &OstreeDeployAction{}
	case "ostree-mirror":
		a.Action = &OstreeMirrorAction{}
	case "ostree-pull":
		a.Action = &OstreePullAction{}
	case "run":
		a.Action = &RunAction{}
	case "template":
		a.Action = &TemplateAction{}
	case "":
		return fmt.Errorf("action name not specified")
	default:
		return fmt.Errorf("unknown action \"%s\"", action.Action)
	}

	if err := unmarshall(a.Action); err != nil {
		return err
	}

	return nil
}

type variables struct {
	KernelArguments []string `yaml:"kernelArguments"`
}

// Manifest represents a manifest
type Manifest struct {
	Variables variables        `yaml:"variables,omitempty"`
	Actions   []ManifestAction `yaml:"actions,omitempty"`
	data      *bytes.Buffer
}

// OpenManifest opens and parses a manifest and returns a Manifest object
func OpenManifest(filePath string, templateVars map[string]interface{}) (*Manifest, error) {
	funcs := template.FuncMap{
		"StringsJoin": strings.Join,
	}

	t, err := template.New(path.Base(filePath)).Funcs(funcs).ParseFiles(filePath)
	if err != nil {
		return nil, err
	}

	data := &bytes.Buffer{}
	if err := t.Execute(data, templateVars); err != nil {
		return nil, err
	}

	var m Manifest

	// Unmarshal
	if err := yaml.Unmarshal(data.Bytes(), &m); err != nil {
		return nil, err
	}

	if len(m.Actions) == 0 {
		return nil, errors.New("no actions declared")
	}

	// Save data
	m.data = data

	return &m, nil
}

// String returns the manifest text after being massaged by the template engine.
func (m *Manifest) String() string {
	return m.data.String()
}
