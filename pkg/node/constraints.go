package node

import "github.com/jonatan5524/own-kubernetes/pkg/agent/api"

const (
	NODE_NAME                = "node"
	NODE_IMAGE               = "own-kube-node"
	NODE_PORT                = api.PORT + "/tcp"
	NODE_PORT_HOST_IP        = "0.0.0.0"
	MEMORY_LIMIT             = 2.9e+9 // 2900MB
	CPU_LIMIT                = 2
	NODE_HOST_MIN_PORT       = 10250
	NODE_HOST_MAX_PORT       = 10300
	NODE_DOCKER_NETWORK_NAME = "kube-net"
	NODE_CIDR                = "172.18.0.0/24"
)
