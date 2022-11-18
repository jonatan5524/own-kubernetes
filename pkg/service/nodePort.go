package service

import (
	"fmt"

	"github.com/jonatan5524/own-kubernetes/pkg/net"
)

const (
	NODE_PORT_SERVICE_CHAIN = "KUBE-NODEPORT"
)

func NewNodePortService(port string, podsCidr string, nodeIP string) (*Service, *Service, error) {
	clusterIPService, err := NewClusterIPService("3001", podsCidr)
	if err != nil {
		return nil, nil, err
	}

	// iptables -t nat -N KUBE-NODEPORTS
	if err := net.NewIPTablesChain(NODE_PORT_SERVICE_CHAIN); err != nil {
		return nil, nil, err
	}

	// iptables -t nat -A KUBE-SERVICES -j KUBE-NODEPORTS
	if err := net.AppendNewIPTablesRule(fmt.Sprintf("-j %s", NODE_PORT_SERVICE_CHAIN), KUBE_SERVICES_CHAIN); err != nil {
		return nil, nil, err
	}

	// iptables -t nat -I KUBE-NODEPORTS 1 -p tcp -m tcp --dport port -j KUBE-MARK-MASQ
	if err := net.InsertNewIPTablesRule(fmt.Sprintf("-p tcp -m tcp --dport %s -j %s", port, KUBE_SERVICE_MARK), NODE_PORT_SERVICE_CHAIN, 1); err != nil {
		return nil, nil, err
	}

	// iptables -t nat -A KUBE-NODEPORTS -p tcp -m tcp --dport port -j clusterIPService
	if err := net.AppendNewIPTablesRule(fmt.Sprintf("-p tcp -m tcp --dport %s -j %s", port, clusterIPService.Id), NODE_PORT_SERVICE_CHAIN); err != nil {
		return nil, nil, err
	}

	return clusterIPService, &Service{
		Type:   NodePort,
		Id:     NODE_PORT_SERVICE_CHAIN,
		IpAddr: nodeIP,
		Port:   port,
	}, nil
}
