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

import humanfriendly
import yaml

__all__ = [
    'Manifest',
    'parse_manifest',
]

class Manifest(object):
    class Live(object):
        title = 'Linux'
        product = 'Linux (Live)'
        timeout = 600
        squashfs_compression = 'zstd'
    type = ''
    size = 4*1024*1024*1024
    osname = ''
    main_ref = ''
    refs = []
    remote_url = ''
    extra_kargs = []
    live = Live()


def parse_manifest(filename: str) -> Manifest:
    with open(filename, 'r') as fh:
        manifest = yaml.safe_load(fh)

    m = Manifest()
    m.type = manifest.get('type', m.type)
    m.size = humanfriendly.parse_size(manifest.get('size', str(m.size)))
    m.osname = manifest.get('osname', m.osname)
    m.main_ref = manifest.get('main-ref', m.main_ref)
    m.refs = [str(x) for x in manifest.get('refs', m.refs)]
    m.remote_url = manifest.get('remote-url', m.remote_url)
    m.extra_kargs = manifest.get('extra-kargs', m.extra_kargs)
    m.live.title = manifest.get('live', {}).get('title', m.live.title)
    m.live.product = manifest.get('live', {}).get('product', m.live.product)
    m.live.timeout = manifest.get('live', {}).get('timeout', m.live.timeout)
    m.live.squashfs_compression = manifest.get('live', {}).get('squashfs-compression', m.live.squashfs_compression)

    if m.type not in ('live', 'disk', 'qemu', 'vmware', 'metal'):
        raise SystemExit('Invalid image type')

    return m
