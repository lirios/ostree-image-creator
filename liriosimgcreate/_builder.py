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
from ._manifest import Manifest
from ._ostree import Tree
from typing import (
    Any,
    Generator,
    List,
    Optional,
)

import contextlib
import os
import shlex
import shutil
import sys
import subprocess
import tempfile

__all__ = [
    'Builder',
]

class Builder(Object):
    def __init__(self, manifest: Manifest, configdir: str, workdir: str, force: Optional[bool] = False, *args: Any, **kwargs: Any):
        """
        Creates a Builder instance.

        Parameters:
        manifest (Manifest): Manifest object
        configdir (str): Path to a directory with additional files to copy
                         into the final image
        workdir (str): Path to a directory where temporary files are stored
        force (bool): Whether a previously created image file is overwritten or not
        """
        Object.__init__(self, *args, **kwargs)
        # Set attributes
        self.manifest = manifest
        self.configdir = os.path.abspath(configdir) if configdir else None
        self.parentdir = os.path.abspath(workdir)
        self.workdir = None
        self.force = force
        # Tree
        self.tmp_repo = os.path.join(workdir, 'repo')
        self.tree = Tree(repo_path=self.tmp_repo, osname=manifest.osname, ref=manifest.main_ref, refs=manifest.refs)

    def initialize(self) -> None:
        """
        Sets up the workspace.
        """
        # Create work directory
        self.print_step('Setting up temporary workspace.')
        self.workdir = tempfile.TemporaryDirectory(dir=self.parentdir, prefix='.mkliriosimage-')
        self.print_step('Temporary workspace in ' + self.workdir.name + ' is now set up.')
        # Clone the repository if needed
        with self.complete_step('Mirroring OSTree repository'):
            try:
                self.tree.mirror(self.manifest.remote_url)
                self.tree.prune()
            except:
                shutil.rmtree(self.tmp_repo)
                raise

    def warning(self, text: str, *args: Any, **kwargs: Any) -> None:
        """
        Prints a warning message in yellow.

        Parameters:
        text (str): Format string to print
        *args (any): Format arguments
        **kwargs (any): Named format arguments
        """
        sys.stderr.write('\033[93m' + text.format(*args, **kwargs) + '\033[0m\n')

    def error(self, text: str, *args: Any, **kwargs: Any) -> None:
        """
        Prints an error message in red.

        Parameters:
        text (str): Format string to print
        *args (any): Format arguments
        **kwargs (any): Named format arguments
        """
        sys.stderr.write('\033[91m' + text.format(*args, **kwargs) + '\033[0m\n')

    def print_step(self, text: str) -> None:
        """
        Prints the step name.

        Parameters:
        text (str): Text to print
        """
        sys.stderr.write('‣ \033[0;1;39m' + text + '\033[0m\n')

    def print_running_cmd(self, args: List[str]) -> None:
        """
        Prints the command that is going to be executed.

        Parameters:
        args (list): Arguments list
        """
        sys.stderr.write("‣ \033[0;1;39mRunning command:\033[0m\n")
        sys.stderr.write(" ".join(shlex.quote(x) for x in args) + "\n")

    @contextlib.contextmanager
    def complete_step(self, text: str, text2: Optional[str] = None) -> Generator[List[Any], None, None]:
        """
        Prints the step name and a message when it's complete.
        If the optional complete message is not provided, 'complete' will
        be used.

        Parameters:
        text (str): Step name
        text2 (str) Optional complete message
        """
        self.print_step(text + '...')
        args: List[Any] = []
        yield args
        if text2 is None:
            text2 = text + ' complete'
        self.print_step(text2.format(*args) + '.')

    def _mkfs_fat(self, label: str, filename: str) -> None:
        self.run(['mkfs.fat', '-n', label, filename], check=True)

    def _mkfs_ext4(self, label: str, filename: str) -> None:
        self.run(['mkfs.ext4', '-L', label, filename], check=True)

    def _mount(self, dev: str, path: str, discard: bool = False, loop: bool = False, readonly: bool = False) -> None:
        os.makedirs(path, 0o755, True)
        args = ['mount', '-n', dev, path]
        if discard:
            args.append('-o')
            args.append('discard')
        if loop:
            args.append('-o')
            args.append('loop')
        if readonly:
            args.append('-o')
            args.append('ro')
        self.run(args, check=True)

    def _umount(self, path: str) -> None:
        # Ignore the errors
        self.run(['umount', '--recursive', '-n', path], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)

    def _matchpathcon(self, path: str) -> str:
        c = self.run(['matchpathcon', '-n', path], stdout=subprocess.PIPE, check=True)
        return c.stdout.decode('utf-8').strip()
