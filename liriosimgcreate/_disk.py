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

from ._builder import Builder
from typing import (
    Any,
    Generator,
    List,
    Optional,
)

import contextlib
import os
import sys
import subprocess
import tempfile
import uuid

__all__ = [
    'DiskBuilder'
]

# GPT partition types
GPT_ESP = uuid.UUID('C12A7328-F81F-11D2-BA4B-00A0C93EC93B')
GPT_BIOS = uuid.UUID('21686148-6449-6E6F-744E-656564454649')
GPT_LINUX = uuid.UUID('0FC63DAF-8483-4772-8E79-3D69D8477DE4')

grub_cfg = """set pager=1
search --label boot --set boot
set root=$boot

if [ -f ${config_directory}/grubenv ]; then
  load_env -f ${config_directory}/grubenv
elif [ -s $prefix/grubenv ]; then
  load_env
fi

if [ x"${feature_menuentry_id}" = xy ]; then
  menuentry_id_option="--id"
else
  menuentry_id_option=""
fi

function load_video {
  if [ x$feature_all_video_module = xy ]; then
    insmod all_video
  else
    insmod efi_gop
    insmod efi_uga
    insmod ieee1275_fb
    insmod vbe
    insmod vga
    insmod video_bochs
    insmod video_cirrus
  fi
}

serial --speed=115200
terminal_input serial console
terminal_output serial console
if [ x$feature_timeout_style = xy ] ; then
  set timeout_style=menu
  set timeout=1
# Fallback normal timeout code in case the timeout_style feature is
# unavailable.
else
  set timeout=1
fi

blscfg
"""

class DiskBuilder(Builder):
    def __init__(self, filename: str, *args: Any, **kwargs: Any):
        """
        Creates a disk image builder.

        Parameters:
        filename (str): Disk image file name
        """
        Builder.__init__(self, *args, **kwargs)
        self.filename = os.path.abspath(filename)

    def _determine_partition_table(self) -> str:
        pn = 3
        ptable = 'label: gpt\n'
        ptable += f'size=384MiB, type={GPT_LINUX}, name="boot"\n'
        ptable += f'size=127MiB, type={GPT_ESP}, name="EFI-SYSTEM"\n'
        if self.tree.basearch == 'x86_64':
            ptable += f'size=1MiB, type={GPT_BIOS}, name="BIOS-BOOT"\n'
            pn += 1
        ptable += f'type={GPT_LINUX}, name="root"\n'
        return ptable, pn

    def _install_uefi(self, deployrootdir: str, rootdir: str, efidir: str) -> None:
        # See also https://github.com/ostreedev/ostree/pull/1873#issuecomment-524439883
        # In the future it'd be better to get this stuff out of the OSTree commit and
        # change our build process to download+extract it separately.
        src_efidir = os.path.join(deployrootdir, 'usr/lib/ostree-boot/efi')
        src_efibootdir = os.path.join(src_efidir, 'EFI/BOOT')
        dst_efibootdir = os.path.join(efidir, 'EFI/BOOT')
        os.makedirs(dst_efibootdir, 0o755, True)
        for root, dirs, files in os.walk(src_efibootdir):
            for name in files:
                if name.startswith('BOOT'):
                    self.run(['cp', '-a', '--reflink=never', os.path.join(root, name), dst_efibootdir], check=True)
        vendorid = None
        for root, dirs, files in os.walk(os.path.join(src_efidir, 'EFI')):
            for name in files:
                if name.startswith('grub') and name.endswith('.efi'):
                    filename = os.path.join(root, name)
                    self.run(['cp', '-a', '--reflink=never', filename, dst_efibootdir], check=True)
                    vendorid = os.path.basename(os.path.dirname(filename))
        if vendorid:
            vendordir = os.path.join(efidir, 'EFI', vendorid)
            os.makedirs(vendordir, 0o755, True)
            with open(os.path.join(vendordir, 'grub.cfg'), 'w') as f:
                f.write('search --label boot --set prefix\n')
                f.write('set prefix=($prefix)/grub2\n')
                f.write('normal\n')
        grub2dir = os.path.join(rootdir, 'boot/grub2')
        os.makedirs(grub2dir, 0o755, True)
        with open(os.path.join(grub2dir, 'grub.cfg'), 'w') as f:
            f.write(grub_cfg)

    def _install_grub2(self, loopdev: str, bootdir: str) -> None:
        # And BIOS grub in addition.  See also
        # https://github.com/coreos/fedora-coreos-tracker/issues/32
        self.run(['grub2-install', '--target', 'i386-pc', '--boot-directory', bootdir, loopdev], check=True)

    @contextlib.contextmanager
    def _attach_image_loopback(self, filename: str) -> Generator[Optional[str], None, None]:
        with self.complete_step('Attaching image file',
                                'Attached image file as {}') as output:
            c = self.run(['losetup', '--find', '--show', '--partscan', filename],
                         stdout=subprocess.PIPE, check=True)
            loopdev = c.stdout.decode('utf-8').strip()
            output.append(loopdev)
        try:
            yield loopdev
        finally:
            with self.complete_step('Detaching image file'):
                self.run(['losetup', '--detach', loopdev], check=True)

    @contextlib.contextmanager
    def _mount_image(self, bootdev: str, efidev: str, rootdev: str) -> Generator[List[str], None, None]:
        with self.complete_step('Mounting image'):
            rootdir = os.path.join(self.workdir.name, 'root')
            bootdir = os.path.join(rootdir, 'boot')
            efidir = os.path.join(bootdir, 'efi')
            ostreedir = os.path.join(rootdir, 'ostree')
            self._mount(rootdev, rootdir, discard=True)
            self._mount(bootdev, bootdir)
            self._mount(efidev, efidir)
            os.makedirs(ostreedir, 0o755, True)
        try:
            yield (rootdir, bootdir, efidir, ostreedir)
        finally:
            with self.complete_step('Unmounting image'):
                self._umount(rootdir)

    def build(self):
        """
        Build disk image.
        """

        if os.path.exists(self.filename) and self.force is False:
            self.error(f'Image file {self.filename} already exist.')
            self.error('You can force a rebuild with \'--force\'.')
            return

        # Create an empty image file
        filename = os.path.join(self.workdir.name, os.path.basename(self.filename))
        with open(filename, 'wb') as f:
            f.seek(self.manifest.size - 1)
            f.write(b'\0')

        # Attach the empty image to a loop device
        with self._attach_image_loopback(filename) as loopdev:
            ptable, rootpn = self._determine_partition_table()
            bootdev = loopdev + 'p1'
            efidev = loopdev + 'p2'
            rootdev = loopdev + 'p' + str(rootpn)
            # Create the disk
            self.run(['sfdisk', '--color=never', loopdev], input=ptable.encode('utf-8'), check=True)
            self.run(['sync'])

            # Format the partitions
            with self.complete_step('Formatting EFI system partition'):
                self._mkfs_fat('EFI-SYSTEM', efidev)
            with self.complete_step('Formatting boot partition'):
                self._mkfs_ext4('boot', bootdev)
            with self.complete_step('Formatting root partition'):
                self._mkfs_ext4('root', rootdev)

            # Mount image
            with self._mount_image(bootdev, efidev, rootdev) as (rootdir, bootdir, efidir, ostreedir):
                # Give a SELinux label, FAT doesn't support it so we do it only on boot and root
                self.run(['chcon', self._matchpathcon('/'), rootdir], check=True)
                self.run(['chcon', self._matchpathcon('/boot'), bootdir], check=True)
                self.run(['chcon', self._matchpathcon('/ostree'), ostreedir], check=True)

                # Initialize OSTree
                # TODO: Replace with `ostree admin init-fs --modern`
                # https://github.com/ostreedev/ostree/pull/1894
                repodir = os.path.join(ostreedir, 'repo')
                deploydir = os.path.join(ostreedir, 'deploy')
                for dirname in (repodir, deploydir):
                    os.makedirs(dirname, 0o755, True)
                with self.complete_step('Pulling OS tree'):
                    self.run(['ostree', '--repo=' + repodir, 'init', '--mode=bare'], check=True)
                    self.run(['ostree', 'pull-local', self.tree.repo_path, self.tree.ref,
                              '--repo=' + repodir, '--disable-fsync'], check=True)
                with self.complete_step('Deploying OS tree'):
                    self.run(['ostree', 'admin', 'os-init', self.tree.osname,
                              '--sysroot=' + rootdir], check=True)
                    self.run(['ostree', 'admin', 'deploy', self.tree.ref,
                              '--sysroot=' + rootdir, '--os=' + self.tree.osname], check=True)

                # Find deployment root
                checksum = self.tree.get_commit_checksum()
                deployrootdir = os.path.join(deploydir, self.tree.osname, 'deploy', checksum + '.0')
                if not os.path.exists(deployrootdir):
                    raise SystemExit(f'Deployment directory for commit {checksum} not found')

                # Setup /var
                vardir = os.path.join(deploydir, self.tree.osname, 'var')
                for dirname in ('home', 'log/journal', 'lib/systemd'):
                    os.makedirs(os.path.join(vardir, dirname), 0o755, True)
                homedir = os.path.join(vardir, 'home')
                self.run(['chcon', self._matchpathcon('/home'), homedir], check=True)

                # Setup boot loader
                self._install_uefi(deployrootdir, rootdir, efidir)
                if self.tree.basearch == 'x86_64':
                    self._install_grub2(loopdev, bootdir)
                with self.complete_step('Configuring sysroot.bootloader'):
                    self.run(['ostree', 'config', '--repo=' + repodir, 'set',
                              'sysroot.bootloader', '"none"'], check=True)

                # Immutable bit
                self.run(['chattr', '+i', rootdir], check=True)

                # Trim
                self.run(['fstrim', '-a', '-v'])

                # Move the file where the user expects it to be
                os.rename(filename, self.filename)

                # Change owner to the unprivileged user so that it's not owned by root
                try:
                    if 'SUDO_UID' in os.environ and 'SUDO_GID' in os.environ:
                        os.chown(self.filename, int(os.environ['SUDO_UID']), int(os.environ['SUDO_GID']))
                except:
                    self.error('Unable to change ISO file ownership')

                sys.stderr.write(f'Wrote: {self.filename}\n')
