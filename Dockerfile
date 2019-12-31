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
    groupadd -g 1002 jenkins && \
    useradd -c "Jenkins" -u 1001 -g 1002 -m -G wheel jenkins && \
    echo "%wheel ALL=(ALL) NOPASSWD: ALL" >/etc/sudoers.d/wheel && \
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
USER jenkins
CMD "sudo /usr/bin/mkliriosimage"
