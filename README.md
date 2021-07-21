# CSI QSD
The CSI QSD is a [CSI](https://kubernetes.io/blog/2019/01/15/container-storage-interface-ga/) driver plugin that uses the [qemu-storage-daemon](https://qemu.readthedocs.io/en/latest/tools/qemu-storage-daemon.html) to create local qcow2 images and it exposes them through the vhost-user protocol. This CSI plugin creates a vhost-user.sock and it mounts it inside the container under the PVC path. The CSI QSD plugin is a local storage provider that implies that the workload that requeries the PVC can be scheduled only on a single node where the PV has been bound to the requested PVC.

## Current features
Dynamic provisioning

## TBD
- Volume extention
- Snapshot
- Clone
- Authentication for the grcp calls. The controller and node service should authenticate in order to be able to exectute the methods.
- Smart scheduling

# Architecture
The qemu-storage-daemon is deployed as a DaemonSet and the methods are exposed through the grpc qsd server. The grpc calls can be execute by calling the methods from ip of the node where the local storage is created and the port `4444`.  


![image info](https://github.com/alicefr/csi-qsd/blob/master/pic/csi-qsd.png)
