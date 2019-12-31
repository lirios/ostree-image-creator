mkliriosimage
=============

[![License](https://img.shields.io/badge/license-GPLv3.0-blue.svg)](https://www.gnu.org/licenses/gpl-3.0.html)
[![GitHub release](https://img.shields.io/github/release/lirios/mkliriosimage.svg)](https://github.com/lirios/mkliriosimage)
[![Build Status](https://travis-ci.org/lirios/mkliriosimage.svg?branch=master)](https://travis-ci.org/lirios/mkliriosimage)
[![GitHub issues](https://img.shields.io/github/issues/lirios/mkliriosimage.svg)](https://github.com/lirios/mkliriosimage/issues)

Tool to build Liri OS images.

This program uses Rust logging that you can configure with the `RUST_LOG`
environment variable, please check [this](https://doc.rust-lang.org/1.1.0/log/index.html) out.

## Dependencies

```sh
sudo dnf install -y \
    coreutils \
    util-linux \
    ostree \
    syslinux \
    syslinux-nonlinux \
    genisoimage \
    xorriso \
    isomd5sum \
    squashfs-tools \
    grub2
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
