<!--
SPDX-FileCopyrightText: 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>

SPDX-License-Identifier: GPL-3.0-or-later
-->

ostree-image-creator
====================

[![License](https://img.shields.io/badge/license-GPLv3.0-blue.svg)](https://www.gnu.org/licenses/gpl-3.0.html)
[![GitHub release](https://img.shields.io/github/release/lirios/ostree-image-creator.svg)](https://github.com/lirios/ostree-image-creator)
[![CI](https://github.com/lirios/ostree-image-creator/workflows/CI/badge.svg?branch=develop)](https://github.com/lirios/ostree-image-creator/actions?query=workflow%3ACI)
[![GitHub issues](https://img.shields.io/github/issues/lirios/ostree-image-creator.svg)](https://github.com/lirios/ostree-image-creator/issues)

ostree-image-creator, or `oic`, is a tool to create live and disk images
for an OSTree-based operating system.

`oic` provides the following subcommands:

 * **resolve**: expands variables and expressions in the manifest file
   and prints it to the screen.
 * **build**: builds an image.

## Dependencies

You need Go installed.

On Fedora:

```sh
sudo dnf install -y golang
```

This programs also use the OSTree library:

```sh
sudo dnf install -y ostree-devel
```

And the following tools to make images:

```sh
sudo dnf install -y \
    coreutils \
    util-linux \
    e2fsprogs \
    dosfstools \
    ostree \
    genisoimage \
    xorriso \
    isomd5sum \
    syslinux \
    squashfs-tools \
    grub2
```

Download all the Go dependencies:

```sh
go mod download
```

## Build

Build with:

```sh
make
```

## Install

Install with:

```sh
make install
```

The default prefix is `/usr/local` but you can specify another one:

```sh
make install PREFIX=/usr
```

And you can also relocate the binaries, this is particularly
useful when building packages:

```
...

%install
make install DESTDIR=%{buildroot} PREFIX=%{_prefix}

...
```

## Licensing

Licensed under the terms of the GNU General Public License version 3 or,
at your option, any later version.
