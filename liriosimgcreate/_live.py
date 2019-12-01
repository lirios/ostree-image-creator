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
from ._syslinux import SysLinux
from ._grub import Grub
from typing import (
    Any,
    Generator,
    Optional,
)

import contextlib
import os
import sys
import shutil
import subprocess
import tempfile
import time

__all__ = [
    'LiveBuilder'
]

class LiveBuilder(Builder):
    def __init__(self, filename: Optional[str] = None, fslabel: Optional[str] = None, *args: Any, **kwargs: Any):
        """
        Creates a live image builder.

        Parameters:
        filename (str): Disk image file name
        fslabel (str): File system label
        """
        Builder.__init__(self, *args, **kwargs)
        self.volset = '%s-%s-%s' % (self.manifest.osname, time.strftime('%Y%m%d%H%M'), self.tree.basearch)
        if filename is None:
            filename = volset + '.iso'
        self.filename = filename
        if fslabel is None:
            fslabel = self.volset
        # The volume ID can only be 32 bytes
        self.fslabel = fslabel[0:32]
        self.syslinux = SysLinux(title=self.manifest.live.title, product=self.manifest.live.product,
                                 fslabel=self.fslabel, timeout=self.manifest.live.timeout)
        self.grub = Grub(title=self.manifest.live.title, product=self.manifest.live.product,
                         fslabel=self.fslabel)

    def _estimate_directory_size(self, start_path: str = '.', add_percent: int = 5) -> int:
        total_size = 0
        for dirpath, dirnames, filenames in os.walk(start_path):
            for f in filenames:
                fp = os.path.join(dirpath, f)
                if not os.path.islink(fp):
                    total_size += os.path.getsize(fp)
        add_percent_modifier = (100.0 + add_percent) / 100.0
        return int(total_size * add_percent_modifier) + 1

    def _create_rootfs(self, filename: str) -> None:
        """
        Creates root file system image.

        Parameters:
        filename (str): Image file name
        """
        with self.complete_step('Creating root file system'):
            # Create an empty image file
            with open(filename, 'wb') as f:
                f.seek(self.manifest.size - 1)
                f.write(b'\0')

            # Format the root file system
            self.run(['mkfs.ext4', filename], check=True)
            mountpoint = os.path.join(self.workdir.name, 'root')
            os.makedirs(mountpoint, 0o755, True)
            with self._mount_loop(filename, mountpoint):
                # Pull and deploy OS tree
                ostreedir = os.path.join(mountpoint, 'ostree')
                repodir = os.path.join(ostreedir, 'repo')
                deploydir = os.path.join(ostreedir, 'deploy')
                for dirname in (repodir, deploydir):
                    os.makedirs(dirname, 0o755, True)
                with self.complete_step('Pulling OS tree(s)'):
                    self.run(['ostree', '--repo=' + repodir, 'init', '--mode=bare'], check=True)
                    self.run(['ostree', 'pull-local',
                            '--repo=' + repodir, '--disable-fsync',
                            self.tree.repo_path] + self.tree.refs,
                            check=True)
                with self.complete_step('Deploying OS tree'):
                    self.run(['ostree', 'admin', 'os-init', self.tree.osname,
                            '--sysroot=' + mountpoint])
                    self.run(['ostree', 'admin', 'deploy', self.tree.ref,
                            '--sysroot=' + mountpoint,
                            '--os=' + self.tree.osname])

                # Create a few directories under /var and label /var/home to make SELinux happy
                # https://github.com/coreos/ignition-dracut/pull/79#issuecomment-488446949
                vardir = os.path.join(deploydir, self.tree.osname, 'var')
                for dirname in ('home', 'log/journal', 'lib/systemd'):
                    os.makedirs(os.path.join(vardir, dirname), 0o755, True)
                homedir = os.path.join(vardir, 'home')
                self.run(['chcon', self._matchpathcon('/home'), homedir], check=True)

    def _create_efiboot(self, dir_path: str, filename: str) -> None:
        """
        Creates a vfat image with EFI boot files.

        Parameters:
        dir_path (str): Path where EFI files are unpacked
        filename (str): Image file name
        """
        with self.complete_step('Creating EFI boot image'):
            # Estimate directory size
            size = self._estimate_directory_size(dir_path, add_percent=25)

            # Create an empty image file
            with open(filename, 'wb') as f:
                f.seek(size - 1)
                f.write(b'\0')

            self.run(['mkfs.msdos', filename], check=True)
            mountpoint = os.path.join(self.workdir.name, 'efiboot')
            with self._mount_loop(filename, mountpoint):
                destdir = os.path.join(mountpoint, 'EFI')
                os.makedirs(destdir, 0o755, True)
                self.run(['cp', '-R', '-L', '--preserve=timestamps', '.', destdir], cwd=dir_path, check=True)

    @contextlib.contextmanager
    def _mount_loop(self, filename: str, mountpoint: str) -> Generator[None, None, None]:
        self._mount(filename, mountpoint, loop=True)
        try:
            yield
        finally:
            self._umount(mountpoint)

    def build(self):
        """
        Build a live image.
        """

        if os.path.exists(self.filename) and self.force is False:
            self.error(f'Image file {self.filename} already exist.')
            self.error('You can force a rebuild with \'--force\'.')
            return

        # Convention for kernel and initramfs names
        kernel_img = 'vmlinuz'
        initramfs_img = 'initramfs.img'

        # Create work directory
        tmp_isoroot = os.path.join(self.workdir.name, 'live')
        tmp_isoimages = os.path.join(tmp_isoroot, 'images')
        tmp_isolinux = os.path.join(tmp_isoroot, 'isolinux')
        tmp_efidir = os.path.join(tmp_isoroot, 'EFI', 'fedora')
        initramfs = os.path.join(tmp_isoimages, initramfs_img)

        # Create empty directories
        for dirname in (tmp_isoroot, tmp_isoimages, tmp_isolinux, tmp_efidir):
            os.makedirs(dirname, 0o755, True)

        tmp_isofile = os.path.join(self.workdir.name, self.filename)

        # Resolve ref to commit checksum
        commit = self.tree.get_commit_checksum()

        # Find the directory under `/usr/lib/modules/<kver>` where the
        # kernel/initrd live. It will be the 2nd entity output by
        # `ostree ls <commit> /usr/lib/modules`
        moduledir = self.tree.list('/usr/lib/modules', commit)[1]

        # Copy those files from the OS tree to the ISO root dir
        with self.complete_step('Extracting kernel and initramfs'):
            for filename in (kernel_img, initramfs_img):
                self.tree.checkout(os.path.join(moduledir, filename), tmp_isoimages, commit)
                # initramfs isn't world readable by default so let's open up perms
                os.chmod(os.path.join(tmp_isoimages, filename), 0o755)

        # Copy memtest from `/usr/lib/ostree-boot` using a glob because there's an always
        # changing version in the file name
        has_memtest = False
        with self.complete_step('Extracting memtest'):
            memtest_list = self.tree.list('/usr/lib/ostree-boot', commit)
            for filename in memtest_list:
                if os.path.basename(filename).startswith('memtest86+'):
                    has_memtest = True
                    self.tree.checkout(filename, tmp_isoimages, commit)
                    src_path = os.path.join(tmp_isoimages, os.path.basename(filename))
                    dst_path = os.path.join(tmp_isoimages, 'memtest')
                    os.rename(src_path, dst_path)
                    break

        # See if checkisomd5 is available, so we add an entry to the bootloader
        has_checkisomd5 = len(self.tree.list('/usr/bin/checkisomd5', commit)) > 0

        # Create rootfs
        liveosdir = os.path.join(self.workdir.name, 'squashfs', 'LiveOS')
        tmp_diskimage = os.path.join(liveosdir, 'rootfs.img')
        os.makedirs(liveosdir, 0o755, True)
        self._create_rootfs(tmp_diskimage)

        # Compress squashfs
        compression = self.manifest.live.squashfs_compression
        with self.complete_step(f'Compressing squashfs with {compression}'):
            liveosdir = os.path.join(tmp_isoroot, 'LiveOS')
            os.makedirs(liveosdir, 0o755, True)
            squashfs_img = os.path.join(liveosdir, 'squashfs.img')
            self.run(['mksquashfs', '.', squashfs_img, '-comp', compression],
                    check=True,
                    cwd=os.path.relpath(os.path.join(tmp_diskimage, '../..')))

        # Add extra kernel arguments
        kargs_list = self.manifest.extra_kargs or ['quiet', 'rhgb']
        kargs_list.append('rd.live.image')
        kargs = ' '.join(kargs_list)

        # Grab all the contents from the image configuration project
        if self.configdir:
            with self.complete_step('Copying files to ISO'):
                srcdir_prefix = self.configdir + '/'
                for srcdir, dirnames, filenames in os.walk(srcdir_prefix):
                    dir_suffix = srcdir.replace(srcdir_prefix, '', 1)
                    dstdir = os.path.join(tmp_isoroot, dir_suffix)
                    if not os.path.exists(dstdir):
                        os.mkdir(dstdir)
                    for filename in filenames:
                        # Skip development files
                        if filename == 'README-devel.md':
                            continue
                        srcfile = os.path.join(srcdir, filename)
                        dstfile = os.path.join(dstdir, filename)
                        # Assume all files are text
                        with open(srcfile) as fh:
                            buf = fh.read()
                        buf = buf.replace('@@FSLABEL@@', self.fslabel)
                        buf = buf.replace('@@KERNEL-ARGS@@', kargs)
                        with open(dstfile, 'w') as fh:
                            fh.write(buf)
                        shutil.copystat(srcfile, dstfile)
                        print(f'{srcfile} -> {dstfile}')

        # Generate syslinux configuration
        self.syslinux.add_linux_stanza(kargs)
        if has_checkisomd5:
            self.syslinux.add_check_stanza(kargs)
        self.syslinux.set_vesa_stanza(kargs)
        if has_memtest:
            self.syslinux.set_memtest_stanza()
        isolinux = os.path.join(tmp_isolinux, 'isolinux.cfg')
        with open(isolinux, 'w') as fh:
            fh.write(self.syslinux.render())

        # Generate grub configuration
        self.grub.add_linux_stanza(kargs)
        if has_checkisomd5:
            self.grub.add_check_stanza(kargs)
        grub = os.path.join(tmp_efidir, 'grub.cfg')
        with open(grub, 'w') as fh:
            fh.write(self.grub.render())

        # Generate the ISO image. Lots of good info here:
        # https://fedoraproject.org/wiki/User:Pjones/BootableCDsForBIOSAndUEFI
        genisoargs = ['genisoimage', '-verbose',
                    '-V', self.fslabel, '-volset', self.volset,
                    # For  greater portability, consider using both
                    # Joliet and Rock Ridge extensions. Umm, OK :)
                    '-rational-rock', '-J', '-joliet-long']

        # For x86_64 legacy boot (BIOS) booting
        if self.tree.basearch == 'x86_64':
            # Install binaries from syslinux package
            isolinuxfiles = [('/usr/share/syslinux/isolinux.bin', 0o755),
                            ('/usr/share/syslinux/ldlinux.c32',  0o755),
                            ('/usr/share/syslinux/libcom32.c32', 0o755),
                            ('/usr/share/syslinux/libutil.c32',  0o755),
                            ('/usr/share/syslinux/vesamenu.c32', 0o755)]
            with self.complete_step('Copying syslinux files to ISO'):
                for src, mode in isolinuxfiles:
                    dst = os.path.join(tmp_isolinux, os.path.basename(src))
                    shutil.copyfile(src, dst)
                    os.chmod(dst, mode)

            # for legacy bios boot AKA eltorito boot
            genisoargs += ['-eltorito-boot', 'isolinux/isolinux.bin',
                        '-eltorito-catalog', 'isolinux/boot.cat',
                        '-no-emul-boot',
                        '-boot-load-size', '4',
                        '-boot-info-table']

        # For x86_64 and aarch64 UEFI booting
        if self.tree.basearch in ('x86_64', 'aarch64'):
            # Create the efiboot.img file. This is a fat32 formatted
            # filesystem that contains all the files needed for EFI boot
            # from an ISO.
            imageefidir = os.path.join(self.workdir.name, 'efi')
            efibootfile = os.path.join(tmp_isoimages, 'efiboot.img')
            with self.complete_step('Extracting EFI files'):
                self.tree.checkout('/usr/lib/ostree-boot/efi/EFI', imageefidir, commit)
                self._create_efiboot(imageefidir, efibootfile)

            genisoargs += ['-eltorito-alt-boot',
                        '-efi-boot', os.path.relpath(efibootfile, tmp_isoroot),
                        '-no-emul-boot']

        # Define inputs and outputs
        genisoargs += ['-o', tmp_isofile, tmp_isoroot]

        # Make ISO
        with self.complete_step('Creating ISO image'):
            self.run(genisoargs, check=True)

        # Add MBR for x86_64 legacy (BIOS) boot when ISO is copied to a USB stick
        if self.tree.basearch == 'x86_64':
            with self.complete_step('Running isohybrid'):
                self.run(['isohybrid', tmp_isofile], check=True)

        # Implant MD5 for checksum check
        with self.complete_step('Implanting MD5 checksum in ISO image'):
            self.run(['implantisomd5', tmp_isofile], check=True)

        # Move the file where the user expects it to be
        os.rename(tmp_isofile, self.filename)

        # Change owner to the unprivileged user so that it's not owned by root
        with self.complete_step('Changing file ownership'):
            try:
                if 'SUDO_UID' in os.environ and 'SUDO_GID' in os.environ:
                    os.chown(self.filename, int(os.environ['SUDO_UID']), int(os.environ['SUDO_GID']))
            except:
                self.error('Unable to change file ownership')

        sys.stderr.write(f'Wrote: {self.filename}\n')
