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

from typing import (
    Any,
    List,
    Optional,
)

import shlex
import subprocess
import sys

__all__ = [
    'Object',
]

class Object(object):
    def __init__(self, debug: Optional[List[str]] = None):
        """
        Creates an Object instance.

        Parameters:
        debug (list): List of thing to debug
        """
        self.__debug = debug or []

    def run(self, args: List[str], **kwargs: Any) -> subprocess.CompletedProcess:
        """
        Runs a command and prints it if in debug mode.

        Parameters:
        args (list): List of arguments
        **kwargs (any): List of named arguments
        """
        if 'run' in self.__debug:
            sys.stderr.write('+ ' + ' '.join(shlex.quote(x) for x in args) + '\n')
        return subprocess.run(args, **kwargs)
