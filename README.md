# Example how to use the qemu-storage-daemon inside containers

Build containerized version of the qsd:
```bash
$ docker build -t qsd/qsd .
```

Create a development container `qemu-utils` with all the tools we need
```bash 
$ docker build -t qsd/qemu-utils -f Dockerfile.utils .
```
Containerization:
The volume `vol-sharing` is the bind mount to share and communicate with the other containers
```bash
$ docker volume create vol-sharing 
# Launch the qemu-storage-daemon container with unix socket
$ docker run -d --name qemu-storage-daemon -v vol-sharing:/share qsd/qsd
62a7e9117798ee81e8318bbe2c945b92f2cff04859a6cd064343fec16a3cd083
# Connect to qmp
$ docker run --rm -ti -v vol-sharing:/share  qsd/qemu-utils 
rlwrap -C qmp socat STDIN UNIX:/share/qmp.sock
# Create a qcow image
 {"execute": "qmp_capabilities" }
 {"execute": "blockdev-create", "arguments": {"job-id": "job0", "options": {"driver": "file", "filename": "/share/test.qcow2", "size": 0}}}
 {"execute": "job-dismiss", "arguments": {"id": "job0"}}
 {"execute": "blockdev-add", "arguments": {"driver": "file", "filename": "/share/test.qcow2", "node-name": "imgfile"}}
 {"execute": "blockdev-create", "arguments": {"job-id": "job0", "options": {"driver": "qcow2", "file": "imgfile", "size": 134217728}}}
 {"execute": "job-dismiss", "arguments": {"id": "job0"}}
 
 # Export ndb
 {"execute":"nbd-server-add", "arguments":{"device":"imgfile", "name":"test-qcow", "writable":true, "description":"text exporter"}}
 

```

You can also launch qsd with tcp:
```bash
docker run -p 4444:4444 -tid --name qemu-storage-daemon -v vol-sharing:/share qsd/qsd --nbd-server addr.type=unix,addr.path=/share/nbd.sock  \
--chardev socket,host=0.0.0.0,port=4444,server=on,nowait,id=chardev0 --monitor chardev=chardev0

# Connect 
$ telnet localhost 4444

(Ctrl+5 to exit)
```

Start qemu inside the container with test-qcow exporter:
 In `test-fedora`, I have a qcow image with already installed fedora:
 ```bash
 $ docker run --device /dev/kvm:/dev/kvm -v vol-sharing:/share --security-opt label=disable -ti -v $(pwd)/test-fedora:/images  qsd/qemu-utils /bin/sh -c \
 'qemu-system-x86_64   -m 1024 -serial mon:stdio -hda /images/fedora.qcow2 -nographic -cpu host -enable-kvm \
 --blockdev driver=nbd,export=test-qcow,server.type=unix,server.path=/share/nbd.sock,node-name=blockdev0 \
 --device virtio-blk-pci,drive=blockdev0'

 ```

## Capabilities
Capabilities allowed by default for the containers:
 ```bash
 $ docker exec -ti qemu-storage-daemon capsh --print
Current: = cap_chown,cap_dac_override,cap_fowner,cap_fsetid,cap_kill,cap_setgid,cap_setuid,cap_setpcap,cap_net_bind_service,cap_net_raw,cap_sys_chroot,cap_mknod,cap_audit_write,cap_setfcap+eip
Bounding set =cap_chown,cap_dac_override,cap_fowner,cap_fsetid,cap_kill,cap_setgid,cap_setuid,cap_setpcap,cap_net_bind_service,cap_net_raw,cap_sys_chroot,cap_mknod,cap_audit_write,cap_setfcap
Ambient set =
Securebits: 00/0x0/1'b0
 secure-noroot: no (unlocked)
 secure-no-suid-fixup: no (unlocked)
 secure-keep-caps: no (unlocked)
 secure-no-ambient-raise: no (unlocked)
uid=0(root)
gid=0(root)
groups=

 ```