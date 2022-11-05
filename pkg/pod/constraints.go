package pod

const (
	SOCKET_PATH                  = "/run/containerd/containerd.sock"
	NAMESPACE                    = "own-kubernetes"
	LOGS_PATH                    = "/var/log/pods/"
	BRIDGE_NAME                  = "br0"
	POD_CIDR                     = "10.0.x.0/24"
	NODE_LOCAL_NETWORK_INTERFACE = "eth0"
	VXLAN_NAME                   = "vxlan10"
)
