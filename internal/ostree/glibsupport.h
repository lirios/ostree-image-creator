// SPDX-FileCopyrightText: 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
//
// SPDX-License-Identifier: LGPL-3.0-or-later

#pragma once

#include <glib.h>
#include <stdio.h>

// Error

static char *_g_error_get_message(GError *error) {
  g_assert(error != NULL);
  return error->message;
}

// Hash table

static gboolean _g_hash_table_iter_next_variant(GHashTableIter *iter,
                                                GVariant **key,
                                                GVariant **value) {
  g_assert(iter != NULL);
  return g_hash_table_iter_next(iter, (gpointer)key, (gpointer)value);
}

// Variant builder

static void _g_variant_builder_add_pair(GVariantBuilder *builder, char *key,
                                        GVariant *value) {
  g_assert(builder != NULL);
  g_assert(key != NULL);
  g_assert(value != NULL);
  g_variant_builder_add(builder, "{sv}", key, value);
}

// Variant

static const GVariantType *_g_variant_type(char *type) {
  return G_VARIANT_TYPE(type);
}

static void _g_variant_get_su(GVariant *v, const char **checksum,
                              OstreeObjectType *objectType) {
  g_assert(v != NULL);
  g_variant_get(v, "(su)", checksum, objectType);
}

// Misc

static const char *_g_strdup(gpointer string) { return g_strdup(string); }

// Repo

static gboolean _ostree_repo_file_ensure_resolved(GFile *file) {
  return ostree_repo_file_ensure_resolved((OstreeRepoFile *)file, NULL);
}

static OstreeRepoFile *_ostree_repo_file(GFile *file) {
  return OSTREE_REPO_FILE(file);
}

static void _ostree_repo_append_pull_flags(OstreeRepoPullFlags *flags,
                                           int flag) {
  *flags |= flag;
}

static void _pull_cb(OstreeAsyncProgress *self, gpointer user_data) {}

static OstreeAsyncProgress *_ostree_async_progress_new() {
  return ostree_async_progress_new_and_connect(
      ostree_repo_pull_default_console_progress_changed, NULL);
}
