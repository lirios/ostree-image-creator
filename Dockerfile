FROM fedora:31

COPY . /app
RUN set -ex && \
    dnf install -y --setopt='tsflags=' --nodocs \
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
        grub2 && \
    cd /app && \
    ./setup.py build && \
    ./setup.py install --prefix=/usr
ENTRYPOINT ["mkliriosimage"]
