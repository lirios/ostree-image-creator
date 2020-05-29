// SPDX-FileCopyrightText: 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
//
// SPDX-License-Identifier: LGPL-3.0-or-later

package ostree

import (
	"errors"
	"os"
	"path/filepath"
	"unsafe"
)

// #cgo pkg-config: ostree-1
// #include <glib.h>
// #include <ostree.h>
// #include "glibsupport.h"
import "C"

var (
	errEmptyPath   = errors.New("empty path")
	errUnknownMode = errors.New("unknown repository mode")
)

// Repo represents a local ostree repository
type Repo struct {
	path string
	ptr  unsafe.Pointer
}

// OpenRepo attempts to open the repo at the given path
func OpenRepo(path string) (*Repo, error) {
	if path == "" {
		return nil, errEmptyPath
	}

	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	cRepoPath := C.g_file_new_for_path(cPath)
	defer C.g_object_unref(C.gpointer(cRepoPath))

	cRepo := C.ostree_repo_new(cRepoPath)
	if cRepo == nil {
		return nil, errors.New("failed to open repository")
	}

	repo := &Repo{path, unsafe.Pointer(cRepo)}

	var errC *C.GError
	if C.ostree_repo_open(cRepo, nil, &errC) == C.FALSE {
		return nil, convertGError(errC)
	}

	return repo, nil
}

// CreateRepo creates the repository at path.
func CreateRepo(path string, mode RepoMode) (*Repo, error) {
	if path == "" {
		return nil, errEmptyPath
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, err
	}

	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	cRepoPath := C.g_file_new_for_path(cPath)
	defer C.g_object_unref(C.gpointer(cRepoPath))

	cRepo := C.ostree_repo_new(cRepoPath)
	if cRepo == nil {
		return nil, errors.New("failed to open repository")
	}

	cMode := (C.OstreeRepoMode)(C.OSTREE_REPO_MODE_BARE)
	switch mode {
	case RepoModeArchive:
		cMode = (C.OstreeRepoMode)(C.OSTREE_REPO_MODE_ARCHIVE)
	case RepoModeBareUser:
		cMode = (C.OstreeRepoMode)(C.OSTREE_REPO_MODE_BARE_USER)
	case RepoModeBareUserOnly:
		cMode = (C.OstreeRepoMode)(C.OSTREE_REPO_MODE_BARE_USER_ONLY)
	}

	repo := &Repo{path, unsafe.Pointer(cRepo)}

	var cErr *C.GError
	if C.ostree_repo_create(cRepo, cMode, nil, &cErr) == C.FALSE {
		return nil, convertGError(cErr)
	}

	return repo, nil
}

// native converts a Repo struct to its C equivalent.
func (r *Repo) native() *C.OstreeRepo {
	if r.ptr == nil {
		return nil
	}
	return (*C.OstreeRepo)(r.ptr)
}

// Close frees the memory allocated for this object.
func (r *Repo) Close() {
	C.g_object_unref(C.gpointer(r.native()))
}

// Path returns the repository path.
func (r *Repo) Path() string {
	return r.path
}

// GetObjectPath returns the path to the OSTree object passed as argument.
func (r *Repo) GetObjectPath(objectName string) string {
	return filepath.Join(r.path, "objects", objectName[:2], objectName[2:])
}

// GetMode returns the repository mode.
func (r *Repo) GetMode() (RepoMode, error) {
	mode := C.ostree_repo_get_mode(r.native())

	switch mode {
	case C.OSTREE_REPO_MODE_BARE:
		return RepoModeBare, nil
	case C.OSTREE_REPO_MODE_ARCHIVE:
		return RepoModeArchive, nil
	case C.OSTREE_REPO_MODE_BARE_USER:
		return RepoModeBareUser, nil
	case C.OSTREE_REPO_MODE_BARE_USER_ONLY:
		return RepoModeBareUserOnly, nil
	default:
		panic("unknown mode")
	}
}
