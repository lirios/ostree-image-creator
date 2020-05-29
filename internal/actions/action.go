// SPDX-FileCopyrightText: 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
// SPDX-FileCopyrightText: 2017 Collabora Ltd.
//
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"errors"

	"github.com/lirios/ostree-image-creator/internal/oic"
)

// Code slightly based on https://github.com/go-debos/debos
// Copyright (C) 2017 Collabora Ltd.
// Originally licensed under the terms of the Apache License 2.0

// Action is the interface that every action should use
type Action interface {
	String() string
	Description() string

	Validate(context *oic.Context) error
	Run(context *oic.Context) error
	Cleanup(context *oic.Context) error
}

// BaseAction represents an action
type BaseAction struct {
	Action string `yaml:"action"`
	Name   string `yaml:"name,omitempty"`
}

var errNotImplemented = errors.New("not implemented")

// String returns the action name
func (b *BaseAction) String() string {
	return b.Action
}

// Description returns the description of this action, if any
func (b *BaseAction) Description() string {
	if b.Name == "" {
		return b.Action
	}
	return b.Name
}

// Validate verifies the action
func (b *BaseAction) Validate(context *oic.Context) error {
	return errNotImplemented
}

// Run runs the action
func (b *BaseAction) Run(context *oic.Context) error {
	return errNotImplemented
}

// Cleanup cleans up the action
func (b *BaseAction) Cleanup(context *oic.Context) error {
	return errNotImplemented
}
