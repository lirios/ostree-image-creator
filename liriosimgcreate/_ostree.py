#
# This file is part of Liri.
#
# Copyright (C) 2019 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
#
# SPDX-License-Identifier: GPL-3.0-or-later
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with this program.  If not, see <http://www.gnu.org/licenses/>.
#

from ._object import Object
from typing import (
    Any,
    Optional,
)

import gi
gi.require_version('GLib', '2.0')
gi.require_version('OSTree', '1.0')
gi.require_version('RpmOstree', '1.0')
from gi.repository import Gio, GLib, OSTree, RpmOstree

import subprocess

__all__ = [
    'Tree',
]

class Tree(Object):
    class Error(Exception):
        pass

    def __init__(self, repo_path: str, osname: str, ref: str, refs: list, gpg_verify: bool = True,
                 tls_permissive: bool = False, *args: Any, **kwargs: Any):
        Object.__init__(self, *args, **kwargs)
        self.remote_options = {
            'gpg-verify': GLib.Variant('b', gpg_verify),
            'tls-permissive': GLib.Variant('b', tls_permissive),
        }
        self.osname = osname
        self.ref = RpmOstree.varsubst_basearch(ref)
        self.refs = []
        for ref in refs:
            self.refs.append(RpmOstree.varsubst_basearch(ref))
        if self.ref not in self.refs:
            self.refs.append(self.ref)
        self.repo = None
        self.repo_path = repo_path
        self.basearch = RpmOstree.get_basearch()

    def _initialize(self):
        """
        Opens the OSTree repository.
        """
        self.repo = OSTree.Repo.new(Gio.File.new_for_path(self.repo_path))
        if self.repo.open(None) is False:
            raise Tree.Error(
                "Failed to open OSTree repository from '%s'" % self.repo_path)

    def mirror(self, src_url: str):
        """
        Mirrors the remote repository into local path.

        Parameters:
        src_url (str): Remote URL
        """
        self.run(['ostree', '--repo=' + self.repo_path, 'init', '--mode=archive'], check=True)
        self.run(['ostree', '--repo=' + self.repo_path, 'remote', 'add',
                  '--if-not-exists', '--no-gpg-verify', self.osname, src_url], check=True)
        for ref in self.refs:
            self.run(['ostree', '--repo=' + self.repo_path, 'pull', '--mirror', self.osname + ':' + ref], check=True)

    def pull(self):
        """
        Pulls the remote repository.
        """
        self.run(['ostree', '--repo=' + self.repo_path, 'pull', '--mirror', self.osname + ':' + self.ref], check=True)

    def prune(self):
        """
        Prunes all commits older than 1h.
        """
        self.run(['ostree', '--repo=' + self.repo_path, 'prune', '--keep-younger-than=1h'], check=True)

    def checkout(self, src: str, dest: str, commit: str):
        """
        Checks out a file from a commit.

        Parameters:
        src (str): Source path
        dest (str): Destination path
        commit (str): Commit checksum
        """
        self.run(['ostree', 'checkout',
                  '--repo=' + self.repo_path,
                  '--user-mode', '--subpath', src,
                  commit, dest], check=True)

    def list(self, path: str, commit: str):
        """
        Returns the path of a file or directory from a commit, if found.

        Parameters:
        path (str): Path to look for
        commit (str): Commit checksum
        """
        c = self.run(['ostree', 'ls', '--repo=' + self.repo_path,
                      '--nul-filenames-only', commit, path],
                      stdout=subprocess.PIPE, check=True)
        return c.stdout.decode('utf-8').split('\0')

    def get_commit_checksum(self) -> str:
        """
        Returns the checksum of the last commit.

        Returns:
        str: Commit checksum
        """
        if self.repo is None:
            self._initialize()
        return self.repo.resolve_rev(self.ref, True)[1]

    # Derived from coreos-assembler/src/estimate-commit-disk-size.
    # Copyright 2018 Red Hat, Inc
    # Licensed under the new-BSD license (http://www.opensource.org/licenses/bsd-license.php)
    def estimate_disk_size(self, isize: Optional[int] = 512, blksize: Optional[int] = 4096,
                           metadata_overhead_percent: Optional[int] = 5, add_percent: Optional[int] = 15, repo_path: Optional[str] = None):
        """
        Given an OSTree commit, estimate how much disk space it will take.

        Parameters:
        isize (int): inode size (XFS defaults as of RHEL8.0)
        blksize (int): block size (XFS defaults as of RHEL8.0)
        metadata_overhead_percent (int): Metadata overhead percent
        add_percent (int): Additional space percentage to reserve
        """
    
        if repo_path is None:
            repo_path = self.repo_path
        r = OSTree.Repo.new(Gio.File.new_for_path(repo_path))
        if r.open(None) is False:
            raise Tree.Error(
                "Failed to open OSTree repository from '%s'" % self.repo_path)
    
        [_, rev] = r.resolve_rev(self.ref, False)
    
        [_, reachable] = r.traverse_commit(rev, 0, None)
        n_meta = 0
        blks_meta = 0
        n_regfiles = 0
        blks_regfiles = 0
        n_symlinks = 0
        blks_symlinks = 0
        for k, v in reachable.items():
            csum, objtype = k.unpack()
            if objtype == OSTree.ObjectType.FILE:
                [_, _, finfo, _] = r.load_file(csum, None)
                if finfo.get_file_type() == Gio.FileType.REGULAR:
                    n_regfiles += 1
                    sz = finfo.get_size()
                    blks_regfiles += (sz // blksize) + 1
                else:
                    n_symlinks += 1
                    sz = len(finfo.get_symlink_target())
                    blks_symlinks += (sz // blksize) + 1
            else:
                [_, sz] = r.query_object_storage_size(objtype, csum, None)
                n_meta += 1
                blks_meta += (sz // blksize) + 1
    
        mb = 1024 * 1024
        blks_per_mb = mb // blksize
        total_data_mb = (blks_meta + blks_regfiles + blks_symlinks) // blks_per_mb
        n_inodes = n_meta + n_regfiles + n_symlinks
        total_inode_mb = 1 + ((n_inodes * isize) // mb)
        total_mb = total_data_mb + total_inode_mb
        add_percent = metadata_overhead_percent + add_percent
        add_percent_modifier = (100.0 + add_percent) / 100.0
        estimate_mb = int(total_mb * add_percent_modifier) + 1
        res = {
            'meta': {'count': n_meta,
                     'blocks': blks_meta, },
            'regfiles': {'count': n_regfiles,
                         'blocks': blks_regfiles, },
            'symlinks': {'count': n_symlinks,
                         'blocks': blks_symlinks, },
            'inodes': {'count': n_inodes,
                       'mb': total_inode_mb, },
            'estimate-mb': {'base': total_mb,
                            'final': estimate_mb},
        }
        return res
