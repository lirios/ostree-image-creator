// SPDX-FileCopyrightText: 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
//
// SPDX-License-Identifier: LGPL-3.0-or-later

package ostree

// #cgo pkg-config: ostree-1
// #include <glib.h>
// #include <ostree.h>
// #include "glibsupport.h"
import "C"

// Checkout checks out the specified path from the revision rev
func (r *Repo) Checkout(rev, path, destPath string) error {
	var root *C.GFile
	var commitC *C.char
	var errC *C.GError
	if C.ostree_repo_read_commit(r.native(), C.CString(rev), &root, &commitC, nil, &errC) == C.FALSE {
		return convertGError(errC)
	}

	subtree := C.g_file_resolve_relative_path(root, C.CString(path))

	opts := "standard::name,standard::type,standard::size,standard::is-symlink,standard::symlink-target,unix::device,unix::inode,unix::mode,unix::uid,unix::gid,unix::rdev"
	info := C.g_file_query_info(subtree, C.CString(opts), C.G_FILE_QUERY_INFO_NOFOLLOW_SYMLINKS, nil, &errC)
	if info == nil {
		return convertGError(errC)
	}

	dest := C.g_file_new_for_path(C.CString(destPath))
	if C.ostree_repo_checkout_tree(r.native(), C.OSTREE_REPO_CHECKOUT_MODE_USER, 0, dest, C._ostree_repo_file(subtree), info, nil, &errC) == C.FALSE {
		return convertGError(errC)
	}

	return nil
}
