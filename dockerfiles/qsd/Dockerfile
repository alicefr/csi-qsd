FROM fedora:34 as build

RUN dnf update -y && dnf install -y \
  bzip2 \
  diffutils \
  findutils \ 
  gcc \
  git \
  glib2 \
  glib2-devel \
  libaio \
  libaio-devel \
  liburing \
  liburing-devel \
  make \
  meson \
  ninja-build \
  pixman \
  pixman-devel \
  python3 \
  python3-pip \
  && dnf clean all

RUN git clone https://git.qemu.org/git/qemu.git \
    && cd /qemu \
    && mkdir -p build \
    && cd build \
    && ../configure \
       --target-list=x86_64-softmmu \
       --enable-linux-aio \
       --enable-linux-io-uring \
    && make storage-daemon/qemu-storage-daemon
   
FROM fedora:34

RUN dnf update -y && dnf install -y \
  libaio \
  liburing \
  qemu-img \
  procps-ng \
  && dnf clean all

VOLUME /share

COPY --from=build /qemu/build/storage-daemon/qemu-storage-daemon /usr/local/bin
COPY bin/qsd-server /usr/bin/qsd-server
ENTRYPOINT ["/usr/bin/qsd-server"]
CMD ["-port", "4444"]
