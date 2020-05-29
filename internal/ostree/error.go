// SPDX-FileCopyrightText: 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
//
// SPDX-License-Identifier: LGPL-3.0-or-later

package ostree

import "errors"

// #cgo pkg-config: ostree-1
// #include <glib.h>
// #include <ostree.h>
// #include "glibsupport.h"
import "C"

func convertGError(errC *C.GError) error {
	if errC == nil {
		return errors.New("nil GError")
	}

	err := errors.New(C.GoString((*C.char)(C._g_error_get_message(errC))))
	defer C.g_error_free(errC)
	return err
}
