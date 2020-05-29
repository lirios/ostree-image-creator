# SPDX-FileCopyrightText: 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
#
# SPDX-License-Identifier: CC0-1.0

FROM golang:alpine AS build
RUN mkdir /source
COPY . /source/
WORKDIR /source
RUN set -ex && \
    apk --no-cache add ca-certificates build-base make git ostree-dev && \
    go mod download && \
    make && \
    strip bin/oic && \
    mkdir /build && \
    cp bin/oic /build/

FROM fedora:latest
RUN set -ex && \
    dnf install -y --setopt='tsflags=' --nodocs \
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
COPY --from=build /build/oic /usr/local/bin/oic
CMD "/usr/local/bin/oic"
