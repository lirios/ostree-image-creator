// SPDX-FileCopyrightText: 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
//
// SPDX-License-Identifier: LGPL-3.0-or-later

package ostree

// #cgo pkg-config: ostree-1
// #include <glib.h>
// #include <ostree.h>
// #include "glibsupport.h"
import "C"

// WalkFunc is a function called by Walk() for each file
type WalkFunc func(path string) error

func (r *Repo) walkStart(root *C.GFile, path string, walkFn WalkFunc) error {
	f := C.g_file_resolve_relative_path(root, C.CString(path))
	defer C.g_object_unref(C.gpointer(f))

	var errC *C.GError
	opts := "standard::name,standard::type,standard::size,standard::is-symlink,standard::symlink-target,unix::device,unix::inode,unix::mode,unix::uid,unix::gid,unix::rdev"
	info := C.g_file_query_info(f, C.CString(opts), C.G_FILE_QUERY_INFO_NOFOLLOW_SYMLINKS, nil, &errC)
	if info == nil {
		return convertGError(errC)
	}

	if C.g_file_info_get_file_type(info) == C.G_FILE_TYPE_DIRECTORY {
		if _, err := r.walkRecurse(f, 1, walkFn); err != nil {
			return err
		}
	}

	return nil
}

func (r *Repo) walkRecurse(root *C.GFile, depth int, walkFn WalkFunc) (bool, error) {
	var errC *C.GError
	opts := "standard::name,standard::type"
	enumerator := C.g_file_enumerate_children(root, C.CString(opts), C.G_FILE_QUERY_INFO_NOFOLLOW_SYMLINKS, nil, &errC)
	if enumerator == nil {
		return false, convertGError(errC)
	}
	defer C.g_object_unref(C.gpointer(enumerator))

	for {
		info := C.g_file_enumerator_next_file(enumerator, nil, nil)
		if info == nil {
			return false, nil
		}
		defer C.g_object_unref(C.gpointer(info))

		child := C.g_file_enumerator_get_child(enumerator, info)
		defer C.g_object_unref(C.gpointer(child))

		// Call function with the absolute path to the child
		pathC := C.g_file_get_path(child)
		if err := walkFn(C.GoString(pathC)); err != nil {
			return false, err
		}

		if C.g_file_info_get_file_type(info) == C.G_FILE_TYPE_DIRECTORY {
			result, err := r.walkRecurse(child, depth, walkFn)
			if err != nil {
				return false, err
			}
			if !result {
				return false, nil
			}
		}
	}

	return true, nil
}

// Walk walks the path and execute walkFn for each file
func (r *Repo) Walk(rev, path string, walkFn WalkFunc) error {
	var root *C.GFile
	var commitC *C.char
	var errC *C.GError
	if C.ostree_repo_read_commit(r.native(), C.CString(rev), &root, &commitC, nil, &errC) == C.FALSE {
		return convertGError(errC)
	}

	if err := r.walkStart(root, path, walkFn); err != nil {
		return err
	}

	return nil
}
