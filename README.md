mkliriosimage
=============

[![License](https://img.shields.io/badge/license-GPLv3.0-blue.svg)](https://www.gnu.org/licenses/gpl-3.0.html)
[![GitHub release](https://img.shields.io/github/release/lirios/mkliriosimage.svg)](https://github.com/lirios/mkliriosimage)
[![Build Status](https://travis-ci.org/lirios/mkliriosimage.svg?branch=develop)](https://travis-ci.org/lirios/mkliriosimage)
[![GitHub issues](https://img.shields.io/github/issues/lirios/mkliriosimage.svg)](https://github.com/lirios/mkliriosimage/issues)

Tool to build Liri OS images.

!!!!!!!! THIS IS A PROTOTYPE, the repository will be FORCE PUSHED frequently !!!!!!!!!!!!

## Dependencies

```sh
sudo dnf install -y \
    python3-gobject-base \
    python3-humanfriendly \
    rpm-ostree-devel \
    ostree-devel \
    coreutils \
    util-linux \
    genisoimage \
    xorriso \
    isomd5sum \
    squashfs-tools \
    grub2
```

## Installation

Build with:

```sh
./setup.py build
```

Install for all the users in your system:

```sh
./setup.py install # use sudo if necessary
```

Or install for your user account only:

```sh
./setup.py install --user
```

## Licensing

Licensed under the terms of the GNU General Public License version 3 or,
at your option, any later version.
