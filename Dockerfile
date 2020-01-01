FROM rust:1.40-slim AS build
RUN mkdir /source
COPY Cargo* /source/
COPY src/ /source/src
WORKDIR /source
RUN set -ex && \
    cargo build --release && \
    strip target/release/oic && \
    mkdir /build && \
    cp target/release/oic /build/

FROM fedora:31
RUN set -ex && \
    dnf install -y --setopt='tsflags=' --nodocs \
        coreutils \
        util-linux \
        e2fsprogs \
        ostree \
        syslinux \
        syslinux-nonlinux \
        genisoimage \
        xorriso \
        isomd5sum \
        squashfs-tools \
        grub2
COPY --from=build /build/oic /usr/bin
CMD "/usr/bin/oic"
