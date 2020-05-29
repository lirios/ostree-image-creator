// SPDX-FileCopyrightText: 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
//
// SPDX-License-Identifier: LGPL-3.0-or-later

package ostree

import (
	"unsafe"
)

// #cgo pkg-config: ostree-1
// #include <glib.h>
// #include <ostree.h>
// #include "glibsupport.h"
import "C"

// PullFlags represents flags used to pull an OS tree.
type PullFlags struct {
	Mirror            bool
	CommitOnly        bool
	Untrusted         bool
	BareUserOnlyFiles bool
	TrustedHTTP       bool
}

// Pull pulls refs from the remoteName remote.
func (r *Repo) Pull(remoteName string, refs []string, flags PullFlags) error {
	cRemoteName := C.CString(remoteName)
	defer C.free(unsafe.Pointer(cRemoteName))

	cRefs := C.malloc(C.size_t(len(refs)) * C.size_t(unsafe.Sizeof(uintptr(0))))
	goArray := (*[1<<30 - 1]*C.char)(cRefs)
	for index, item := range refs {
		goArray[index] = C.CString(item)
		defer C.free(unsafe.Pointer(goArray[index]))
	}
	defer C.free(unsafe.Pointer(cRefs))

	cFlags := (C.OstreeRepoPullFlags)(C.OSTREE_REPO_PULL_FLAGS_NONE)
	if flags.Mirror {
		C._ostree_repo_append_pull_flags(&cFlags, (C.int)(C.OSTREE_REPO_PULL_FLAGS_MIRROR))
	}
	if flags.CommitOnly {
		C._ostree_repo_append_pull_flags(&cFlags, (C.int)(C.OSTREE_REPO_PULL_FLAGS_COMMIT_ONLY))
	}
	if flags.Untrusted {
		C._ostree_repo_append_pull_flags(&cFlags, (C.int)(C.OSTREE_REPO_PULL_FLAGS_UNTRUSTED))
	}
	if flags.BareUserOnlyFiles {
		C._ostree_repo_append_pull_flags(&cFlags, (C.int)(C.OSTREE_REPO_PULL_FLAGS_BAREUSERONLY_FILES))
	}
	if flags.TrustedHTTP {
		C._ostree_repo_append_pull_flags(&cFlags, (C.int)(C.OSTREE_REPO_PULL_FLAGS_TRUSTED_HTTP))
	}

	cProgress := C._ostree_async_progress_new()

	var cErr *C.GError
	if C.ostree_repo_pull(r.native(), cRemoteName, (**C.char)(cRefs), cFlags, cProgress, nil, &cErr) == C.FALSE {
		return convertGError(cErr)
	}

	C.ostree_async_progress_finish(cProgress)

	return nil
}

// PullOptions represents options used to pull an OS tree.
type PullOptions struct {
	OverrideRemoteName string
	Refs               []string
}

// PullWithOptions pulls refs from remoteName using options.
func (r *Repo) PullWithOptions(remoteName string, options PullOptions) error {
	cRemoteName := C.CString(remoteName)
	defer C.free(unsafe.Pointer(cRemoteName))

	builder := C.g_variant_builder_new(C._g_variant_type(C.CString("a{sv}")))

	if options.OverrideRemoteName != "" {
		cOverrideRemoteName := C.CString(options.OverrideRemoteName)
		defer C.free(unsafe.Pointer(cOverrideRemoteName))

		key := C.CString("override-remote-name")
		defer C.free(unsafe.Pointer(key))

		value := C.g_variant_new_take_string((*C.gchar)(cOverrideRemoteName))

		C._g_variant_builder_add_pair(builder, key, value)
	}

	if len(options.Refs) > 0 {
		cRefs := make([]*C.gchar, len(options.Refs))
		for index, str := range options.Refs {
			cRefs[index] = (*C.gchar)(C.CString(str))
		}

		key := C.CString("refs")
		defer C.free(unsafe.Pointer(key))

		value := C.g_variant_new_strv((**C.gchar)(&cRefs[0]), (C.gssize)(len(cRefs)))

		C._g_variant_builder_add_pair(builder, key, value)

		for index, s := range cRefs {
			cRefs[index] = nil
			C.free(unsafe.Pointer(s))
		}
	}

	cOptions := C.g_variant_builder_end(builder)

	cProgress := C._ostree_async_progress_new()

	var cErr *C.GError
	if C.ostree_repo_pull_with_options(r.native(), cRemoteName, cOptions, cProgress, nil, &cErr) == C.FALSE {
		return convertGError(cErr)
	}

	C.ostree_async_progress_finish(cProgress)

	return nil
}
