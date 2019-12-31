FROM rust:1.40-slim AS build
RUN mkdir /source
COPY Cargo* /source/
COPY src/ /source/src
WORKDIR /source
RUN set -ex && \
    cargo build --release && \
    strip target/release/mkliriosimage && \
    mkdir /build && \
    cp target/release/mkliriosimage /build/

FROM fedora:31
RUN set -ex && \
    dnf install -y --setopt='tsflags=' --nodocs \
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
COPY --from=build /build/mkliriosimage /usr/bin
CMD "/usr/bin/mkliriosimage"
