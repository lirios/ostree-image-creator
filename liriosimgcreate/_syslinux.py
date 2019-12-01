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
    'SysLinux',
]

class Stanza(object):
    name = ''
    label = ''
    help = ''
    kargs = ''

class SysLinux(object):
    def __init__(self, title='Linux', product='Linux (Live)', fslabel='', timeout=600):
        self.__title = title
        self.__product = product
        self.__fslabel = fslabel
        self.__stanzas = []
        self.__vesa_stanza = None
        self.__memtest_stanza = None
        self.__template = f"""default vesamenu.c32
timeout {timeout}

display boot.msg

menu autoboot Starting {title} in # second{{,s}}. Press any key to interrupt.

# Clear the screen when exiting the menu, instead of leaving the menu displayed.
# For vesamenu, this means the graphical background is still displayed without
# the menu itself for as long as the screen remains in graphics mode.
menu clear
menu background splash.png
menu title {title}
menu vshift 8
menu rows 18
menu margin 8
#menu hidden
menu helpmsgrow 15
menu tabmsgrow 13

# Border Area
menu color border * #00000000 #00000000 none

# Selected item
menu color sel 0 #ffffffff #00000000 none

# Title bar
menu color title 0 #ff7ba3d0 #00000000 none

# Press [Tab] message
menu color tabmsg 0 #ff3a6496 #00000000 none

# Unselected menu item
menu color unsel 0 #84b8ffff #00000000 none

# Selected hotkey
menu color hotsel 0 #84b8ffff #00000000 none

# Unselected hotkey
menu color hotkey 0 #ffffffff #00000000 none

# Help text
menu color help 0 #ffffffff #00000000 none

# A scrollbar of some type? Not sure.
menu color scrollbar 0 #ffffffff #ff355594 none

# Timeout msg
menu color timeout 0 #ffffffff #00000000 none
menu color timeout_msg 0 #ffffffff #00000000 none

# Command prompt text
menu color cmdmark 0 #84b8ffff #00000000 none
menu color cmdline 0 #ffffffff #00000000 none

# Do not display the actual menu unless the user presses a key. All that is displayed is a timeout message.

menu tabmsg Press Tab for full configuration options on menu items.

menu separator # insert an empty line
menu separator # insert an empty line
@@STANZAS@@
menu separator # insert an empty line

# Troubleshooting submenu
menu begin ^Troubleshooting
  menu title Troubleshooting
@@TROUBLESHOOTING@@
menu separator # insert an empty line

label local
  menu label Boot from ^local drive
  localboot 0xffff

menu separator # insert an empty line
menu separator # insert an empty line

label returntomain
  menu label Return to ^main menu
  menu exit

menu end
"""

    def __get_stanza(self, stanza) -> str:
        name = stanza.name
        label = stanza.label
        kargs = stanza.kargs
        fslabel = self.__fslabel
        return f"""
label {name}
  menu label {label}
  kernel /images/vmlinuz
  append initrd=/images/initramfs.img root=live:CDLABEL={fslabel} {kargs}
"""

    def __get_vesa_stanza(self) -> str:
        name = self.__vesa_stanza.name
        label = self.__vesa_stanza.label
        help = self.__vesa_stanza.help
        kargs = self.__vesa_stanza.kargs
        fslabel = self.__fslabel
        return f"""
label {name}
  menu indent count 5
  menu label {label}
  text help
    {help}
  endtext
  kernel /images/vmlinuz
  append initrd=/images/initramfs.img root=live:CDLABEL={fslabel} {kargs} nomodeset
"""

    def __get_memtest_stanza(self) -> str:
        name = self.__memtest_stanza.name
        label = self.__memtest_stanza.label
        help = self.__memtest_stanza.help
        return f"""
label {name}
  menu label {label}
  text help
    {help}
  endtext
  kernel /images/memtest
"""

    def add_linux_stanza(self, kargs: str = ''):
        stanza = Stanza()
        stanza.name = 'linux'
        stanza.label = '^Start ' + self.__product
        stanza.kargs = kargs
        self.__stanzas.append(stanza)

    def add_check_stanza(self, kargs: str = ''):
        stanza = Stanza()
        stanza.name = 'check'
        stanza.label = 'Test this ^media & start ' + self.__product
        stanza.kargs = kargs + ' rd.live.check'
        self.__stanzas.append(stanza)

    def set_vesa_stanza(self, kargs: str = ''):
        product = self.__product
        self.__vesa_stanza = Stanza()
        self.__vesa_stanza.name = 'vesa'
        self.__vesa_stanza.label = f'Start {product} in ^basic graphics mode'
        self.__vesa_stanza.help = f'Try this option out if you\'re having trouble starting\n    {product}'
        self.__vesa_stanza.kargs = kargs

    def set_memtest_stanza(self):
        self.__memtest_stanza = Stanza()
        self.__memtest_stanza.name = 'memset'
        self.__memtest_stanza.label = 'Run a ^memory test'
        self.__memtest_stanza.help = 'If your system is having issues, a problem with your\n    ' \
                                     'system\'s memory may be the cause. Use this utility to\n    ' \
                                     'see if the memory is working correctly.'

    def render(self) -> str:
        stanzas = ''
        for stanza in self.__stanzas:
            stanzas += self.__get_stanza(stanza)
        troubleshooting = ''
        if self.__vesa_stanza:
            troubleshooting += self.__get_vesa_stanza()
        if self.__memtest_stanza:
            troubleshooting += self.__get_memtest_stanza()
        return self.__template.replace('@@STANZAS@@', stanzas).replace('@@TROUBLESHOOTING@@', troubleshooting)
