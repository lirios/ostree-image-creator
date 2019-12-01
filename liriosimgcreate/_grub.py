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

__all__ = [
    'Grub',
]

class Stanza(object):
    name = ''
    label = ''
    help = ''
    kargs = ''

class Grub(object):
    def __init__(self, title: str = 'Linux', product: str = 'Linux (Live)', fslabel: str = ''):
        self.__title = title
        self.__product = product
        self.__fslabel = fslabel
        self.__stanzas = []
        self.__template = """set default="1"

function load_video {
  insmod efi_gop
  insmod efi_uga
  insmod video_bochs
  insmod video_cirrus
  insmod all_video
}

load_video
set gfxpayload=keep
insmod gzio
insmod part_gpt
insmod ext2

set timeout=1

@@STANZAS@@
"""

    def __get_stanza(self, stanza) -> str:
        label = stanza.label
        kargs = stanza.kargs
        fslabel = self.__fslabel
        return f"""menuentry '{label}' --class fedora --class gnu-linux --class gnu --class os {{
  linux /images/vmlinuz root=live:CDLABEL={fslabel} {kargs}
  initrd /images/initramfs.img
}}"""

    def add_linux_stanza(self, kargs: str = ''):
        stanza = Stanza()
        stanza.label = 'Start ' + self.__product
        stanza.kargs = kargs
        self.__stanzas.append(stanza)

    def add_check_stanza(self, kargs: str = ''):
        stanza = Stanza()
        stanza.label = 'Test this media & start ' + self.__product
        stanza.kargs = kargs + ' rd.live.check'
        self.__stanzas.append(stanza)

    def render(self) -> str:
        stanzas = ''
        index = 0
        for stanza in self.__stanzas:
            if index > 0:
                stanzas += '\n\n'
            stanzas += self.__get_stanza(stanza)
            index += 1
        return self.__template.replace('@@STANZAS@@', stanzas)
