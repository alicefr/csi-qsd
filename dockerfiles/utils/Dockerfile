FROM fedora:34

RUN dnf update -y && dnf install -y \
	qemu-img \
	qemu-system-x86 \
	socat \
        rlwrap \
   && dnf clean all \
   && rm -rf /var/cache/dnf

COPY qemu-qmp.conf /qemu-qmp.conf
