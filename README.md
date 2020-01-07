ostree-image-creator
====================

[![License](https://img.shields.io/badge/license-GPLv3.0-blue.svg)](https://www.gnu.org/licenses/gpl-3.0.html)
[![GitHub release](https://img.shields.io/github/release/lirios/ostree-image-creator.svg)](https://github.com/lirios/ostree-image-creator)
[![Build Status](https://travis-ci.org/lirios/ostree-image-creator.svg?branch=develop)](https://travis-ci.org/lirios/ostree-image-creator)
[![GitHub issues](https://img.shields.io/github/issues/lirios/ostree-image-creator.svg)](https://github.com/lirios/ostree-image-creator/issues)

ostree-image-creator, or `oic`, is a tool to create live and disk images
for an OSTree-based operating system.

This program uses Rust logging that you can configure with the `RUST_LOG`
environment variable, please check [this](https://doc.rust-lang.org/1.1.0/log/index.html) out.

If you want to trace which commands are executed, set `RUST_LOG=trace`.

## Dependencies

```sh
sudo dnf install -y \
    coreutils \
    util-linux \
    e2fsprogs \
    dosfstools \
    ostree \
    syslinux \
    syslinux-nonlinux \
    genisoimage \
    xorriso \
    isomd5sum \
    squashfs-tools \
    grub2
```

If SELinux is enabled in your manifest file, you need to install the tools:

```sh
sudo dnf install -y \
    libselinux-utils
```

## Installation

Build with:

```sh
cargo build
```

Run with (it needs root privileges):

```sh
sudo cargo run
```

## Licensing

Licensed under the terms of the GNU General Public License version 3 or,
at your option, any later version.
