// SPDX-FileCopyrightText: 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
//
// SPDX-License-Identifier: LGPL-3.0-or-later

package ostree

import (
	"errors"
	"fmt"
)

// #cgo pkg-config: ostree-1
// #include <glib.h>
// #include <ostree.h>
// #include "glibsupport.h"
import "C"

// RepoMode represents a repository mode.
type RepoMode int

const (
	// RepoModeBare indicates that files are stored as themselves.
	// Checkouts are hardlinks and can only be written as root.
	RepoModeBare RepoMode = iota
	// RepoModeArchive indicates that files are compressed.
	// Should be owned by non-root and can be served via HTTP.
	RepoModeArchive
	// RepoModeBareUser indicates that files are stored as themselves
	// except ownership.  It can be written by user.
	// Hardlinks work only in user checkouts.
	RepoModeBareUser
	// RepoModeBareUserOnly is the same as RepoModeBareUser, but all metadata
	// is not stored, so it can only be used for user checkouts.
	// Does not need xattrs.
	RepoModeBareUserOnly
)

// ListRefs lists all the refs in the repository
func (r *Repo) ListRefs() ([]string, error) {
	if r.ptr == nil {
		return nil, errors.New("repo not initialized")
	}

	var refsC *C.GHashTable
	var errC *C.GError
	if C.ostree_repo_list_refs(r.native(), nil, &refsC, nil, &errC) == C.FALSE {
		return nil, convertGError(errC)
	}

	var iter C.GHashTableIter
	C.g_hash_table_iter_init(&iter, refsC)

	refs := []string{}

	var hkey C.gpointer
	var hvalue C.gpointer
	for C.g_hash_table_iter_next(&iter, &hkey, &hvalue) == C.TRUE {
		refs = append(refs, C.GoString(C._g_strdup(hkey)))
	}

	return refs, nil
}

// ListRevisions returns a dictionary whose keys are refs and values are the corresponding revisions
func (r *Repo) ListRevisions() (map[string]string, error) {
	refs, err := r.ListRefs()
	if err != nil {
		return nil, err
	}

	revs := map[string]string{}

	for _, ref := range refs {
		rev, err := r.ResolveRev(ref)
		if err != nil {
			return nil, err
		}
		revs[ref] = rev
	}

	return revs, nil
}

// GetParentRev returns the revision of the parent commit, or an empty string if it doesn't have one
func (r *Repo) GetParentRev(rev string) (string, error) {
	if r.ptr == nil {
		return "", errors.New("repo not initialized")
	}

	var variantC *C.GVariant
	var errC *C.GError
	if C.ostree_repo_load_variant_if_exists(r.native(), C.OSTREE_OBJECT_TYPE_COMMIT, C.CString(rev), &variantC, &errC) == C.FALSE {
		return "", convertGError(errC)
	}
	if variantC == nil {
		return "", fmt.Errorf("commit %s doesn't exist", rev)
	}
	return C.GoString(C.ostree_commit_get_parent(variantC)), nil
}

// ResolveRev returns the revision corresponding to the specified branch
func (r *Repo) ResolveRev(branch string) (string, error) {
	if r.ptr == nil {
		return "", errors.New("repo not initialized")
	}

	var revC *C.char
	var errC *C.GError
	if C.ostree_repo_resolve_rev(r.native(), C.CString(branch), C.FALSE, &revC, &errC) == C.FALSE {
		return "", convertGError(errC)
	}

	return C.GoString(revC), nil
}

// TraverseCommit returns an hash table with all the reachable objects from
// the passed commit checksum, traversing maxDepth parent commits
func (r *Repo) TraverseCommit(rev string, maxDepth int) ([]string, error) {
	if r.ptr == nil {
		return nil, errors.New("repo not initialized")
	}

	revC := C.CString(rev)
	maxDepthC := C.int(maxDepth)

	var creachable *C.GHashTable
	var errC *C.GError
	if C.ostree_repo_traverse_commit(r.native(), revC, maxDepthC, &creachable, nil, &errC) == C.FALSE {
		return nil, convertGError(errC)
	}

	var iter C.GHashTableIter
	C.g_hash_table_iter_init(&iter, creachable)

	objects := []string{}

	var object *C.GVariant
	for C._g_hash_table_iter_next_variant(&iter, &object, nil) == C.TRUE {
		var cchecksum *C.char
		var cobjectType C.OstreeObjectType
		C._g_variant_get_su(object, &cchecksum, &cobjectType)

		objectNameC := C.ostree_object_to_string(cchecksum, cobjectType)
		objectName := C.GoString(objectNameC)

		if cobjectType == C.OSTREE_OBJECT_TYPE_FILE {
			if mode, _ := r.GetMode(); mode == RepoModeArchive {
				// Append z for archive repositories
				objects = append(objects, fmt.Sprintf("%sz", objectName))
				continue
			}
		}

		objects = append(objects, objectName)
	}

	return objects, nil
}

// Prune prunes the repository
func (r *Repo) Prune(noPrune, onlyRefs bool) (int, int, uint64, error) {
	if r.ptr == nil {
		return 0, 0, 0, errors.New("repo not initialized")
	}

	var flags C.OstreeRepoPruneFlags = C.OSTREE_REPO_PRUNE_FLAGS_NONE
	if noPrune {
		flags |= C.OSTREE_REPO_PRUNE_FLAGS_NO_PRUNE
	}
	if onlyRefs {
		flags |= C.OSTREE_REPO_PRUNE_FLAGS_REFS_ONLY
	}

	var total C.gint
	var pruned C.gint
	var size C.guint64
	var errC *C.GError
	if C.ostree_repo_prune(r.native(), flags, -1, &total, &pruned, &size, nil, &errC) == C.FALSE {
		return 0, 0, 0, convertGError(errC)
	}

	return int(total), int(pruned), uint64(size), nil
}

// SetRefImmediate points ref to checksum for the specified remote
func (r *Repo) SetRefImmediate(remote, ref, checksum string) error {
	if r.ptr == nil {
		return errors.New("repo not initialized")
	}

	var remoteC *C.char
	if remote != "" {
		remoteC = C.CString(remote)
	}

	var errC *C.GError
	if C.ostree_repo_set_ref_immediate(r.native(), remoteC, C.CString(ref), C.CString(checksum), nil, &errC) == C.FALSE {
		return convertGError(errC)
	}

	return nil
}

// RegenerateSummary updates the summary
func (r *Repo) RegenerateSummary() error {
	if r.ptr == nil {
		return errors.New("repo not initialized")
	}

	var errC *C.GError
	if C.ostree_repo_regenerate_summary(r.native(), nil, nil, &errC) == C.FALSE {
		return convertGError(errC)
	}

	return nil
}
