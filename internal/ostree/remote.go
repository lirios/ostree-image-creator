// SPDX-FileCopyrightText: 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
//
// SPDX-License-Identifier: LGPL-3.0-or-later

package ostree

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"unicode"
	"unsafe"
)

// #cgo pkg-config: ostree-1
// #include <glib.h>
// #include <ostree.h>
// #include "glibsupport.h"
import "C"

// Partially based on code from https://github.com/sjoerdsimons/ostree-go
// Copyright 2017, Collabora Ltd.

// RemoteOptions represents the options to be used when adding a remote.
type RemoteOptions struct {
	ContentURL          string "contenturl"
	Proxy               string
	NoGPGVerify         bool "gpg-verify,invert"
	NoGPGVerifySummary  bool "gpg-verify-summary,invert"
	TLSPermissive       bool
	TLSClientCertPath   string
	TLSClientKeyPath    string
	TLSCaPath           string
	UnconfiguredState   string
	MinFreeSpacePercent string
	CollectionID        string
}

func toDashString(in string) string {
	var out bytes.Buffer
	for i, c := range in {
		if !unicode.IsUpper(c) {
			out.WriteRune(c)
			continue
		}

		if i > 0 {
			out.WriteString("-")
		}
		out.WriteRune(unicode.ToLower(c))
	}

	return out.String()
}

func convertRemoteOptions(options RemoteOptions) *C.GVariant {
	casv := C.CString("a{sv}")
	defer C.free(unsafe.Pointer(casv))

	builder := C.g_variant_builder_new(C._g_variant_type(casv))

	v := reflect.ValueOf(options)
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		vf := v.Field(i)
		tf := t.Field(i)
		invert := false

		name := toDashString(tf.Name)
		if tf.Tag != "" {
			opts := strings.Split(string(tf.Tag), ",")
			if opts[0] != "" {
				name = opts[0]
			}
			for _, o := range opts[1:] {
				switch strings.TrimSpace(o) {
				case "invert":
					invert = true
				default:
					panic(fmt.Sprintf("Unhandled flag: %s", o))
				}
			}
		}

		var variant *C.GVariant
		switch vf.Kind() {
		case reflect.Bool:
			/* Should probalby use e.g. Maybe type so it can judge unitialized */
			b := vf.Bool()
			var cb C.gboolean

			if !b {
				// Still the default, so don't bother setting it
				continue
			}
			if invert {
				cb = C.gboolean(0)
			} else {
				cb = C.gboolean(1)
			}
			variant = C.g_variant_new_boolean(cb)
		case reflect.String:
			if vf.String() == "" {
				continue
			}
			variant = C.g_variant_new_take_string((*C.gchar)(C.CString(vf.String())))
		default:
			panic(fmt.Sprintf("Can't handle type of field: %s", tf.Name))
		}

		cName := C.CString(name)
		defer C.free(unsafe.Pointer(cName))
		C._g_variant_builder_add_pair(builder, cName, variant)
	}

	return C.g_variant_builder_end(builder)
}

// RemoteAdd adds remote name from url.
func (r *Repo) RemoteAdd(name, url string, options RemoteOptions) error {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	cURL := C.CString(url)
	defer C.free(unsafe.Pointer(cURL))

	cOptions := convertRemoteOptions(options)
	C.g_variant_ref_sink(cOptions)
	defer C.g_variant_unref(cOptions)

	var cErr *C.GError
	if C.ostree_repo_remote_add(r.native(), cName, cURL, cOptions, nil, &cErr) == C.FALSE {
		return convertGError(cErr)
	}

	return nil
}

// RemoteDelete deletes remote name.
func (r *Repo) RemoteDelete(name string) error {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	var cErr *C.GError
	if C.ostree_repo_remote_delete(r.native(), cName, nil, &cErr) == C.FALSE {
		return convertGError(cErr)
	}

	return nil
}

// HasRemote returns whether there is the remote name.
func (r *Repo) HasRemote(name string) bool {
	var cNumRemotes C.uint
	cList := C.ostree_repo_remote_list(r.native(), &cNumRemotes)
	length := uint(cNumRemotes)
	tmpSlice := (*[1 << 30]*C.char)(unsafe.Pointer(cList))[:length:length]
	for _, s := range tmpSlice {
		if C.GoString(s) == name {
			return true
		}
	}

	return false
}
