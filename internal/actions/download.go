// SPDX-FileCopyrightText: 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/mholt/archiver"

	"github.com/lirios/ostree-image-creator/internal/logger"
	"github.com/lirios/ostree-image-creator/internal/oic"
)

// DownloadAction represents an action called "download"
type DownloadAction struct {
	BaseAction `yaml:",inline"`
	URL        *url.URL `yaml:"url"`
	DestPath   string   `yaml:"filename,omitempty"`
	Unpack     bool     `yaml:"unpack,omitempty"`
	UnpackPath string   `yaml:"unpack-path,omitempty"`
}

// Validate checks if the action is configured correctly
func (a *DownloadAction) Validate(context *oic.Context) error {
	if a.URL.String() == "" {
		return fmt.Errorf("property \"url\" is mandatory for the \"%s\" action", a)
	}

	switch a.URL.Scheme {
	case "http":
	case "https":
		// These are supported
	default:
		return fmt.Errorf("unsupported URL scheme %s, only http and https are supported", a.URL.Scheme)
	}

	if a.Name == "" {
		return fmt.Errorf("property \"name\" is mandatory for the \"%s\" action", a)
	}

	// Make sure files are downloaded in the appropriate place
	if a.DestPath == "" {
		a.DestPath = path.Join(context.DownloadDir, filepath.Base(a.URL.Path))
	} else {
		a.DestPath = path.Join(context.DownloadDir, a.DestPath)
	}

	// Make sure an unpack path is set and it's in the appropriate place
	if a.Unpack {
		if a.UnpackPath == "" {
			a.UnpackPath = path.Join(context.DownloadDir, filepath.Base(a.URL.Path)+".unpack")
		} else {
			a.UnpackPath = path.Join(context.DownloadDir, filepath.Base(a.URL.Path))
		}
	}

	return nil
}

// Run runs the action
func (a *DownloadAction) Run(context *oic.Context) error {
	// Download the file
	if err := download(a.URL.String(), a.DestPath); err != nil {
		return err
	}

	// Unpack
	if a.Unpack {
		return archiver.Unarchive(a.DestPath, a.UnpackPath)
	}

	return nil
}

// Cleanup cleans up after the action
func (a *DownloadAction) Cleanup(context *oic.Context) error {
	// Remove the downloaded file
	if err := os.Remove(a.DestPath); err != nil {
		return err
	}

	// Remove the unpacked files
	if a.Unpack {
		if err := os.RemoveAll(a.UnpackPath); err != nil {
			return err
		}
	}

	return nil
}

// downlod downloads url into destPath
func download(url, destPath string) error {
	logger.Actionf("Downloading \"%s\"...", url)

	// If the destination path already exist and is a regular file
	// we remove it before downloading the file, otherwise if it's
	// a directory something is wrong
	if fileInfo, err := os.Stat(destPath); os.IsExist(err) {
		if fileInfo.Mode().IsRegular() {
			os.Remove(destPath)
		} else if fileInfo.Mode().IsDir() {
			return fmt.Errorf("destination path \"%s\" is a directory", destPath)
		}
	}

	r, err := http.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download \"%s\": %s", url, r.Status)
	}

	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := io.Copy(file, r.Body); err != nil {
		return err
	}

	return nil
}
