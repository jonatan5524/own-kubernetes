# Own-Kubernetes
Hello, I had my work on trying to write on my own Kubernetes like program
It consists of kubelet, kube-proxy and kube-api

## Prerequsuite:
- Golang 1.22+ installed
- Makefile installed
- Docker installed (for building images)
- Qemu installed
- Need to have in the repository VM image for the node VM (I have used debian image) located in ./vms folder - in the image need to be installed containerd
- Need to enable libvirtd
- Need to create ssh keys in and place them in ./vms/.ssh/

## How To run?
```Makefile
  make
```
- This will build the images and upload them to the regisrty
- Build kubelet binary and kubectl like binary
- Start the node
- SSH to it


To start the kubelet in the node:
```bash
 sudo ./kubelet --kubernetes-api-endpoint 'http://localhost:8080'
```

To interact with the kube-api (in local machine):
```bash
export KUBE_API_ENDPOINT=http://${NODE_IP}:8080
./bin/own-kubectl get pods
```
