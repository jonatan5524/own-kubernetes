package service

import (
	"fmt"

	"github.com/jonatan5524/own-kubernetes/pkg"
	"github.com/jonatan5524/own-kubernetes/pkg/net"
)

const (
	KUBE_SERVICES_CHAIN       = "KUBE-SERVICES"
	CLUSTER_IP_SERVICE_PREFIX = "KUBE-SVC"
	KUBE_SERVICE_MARK         = "KUBE-MARK-MASQ"
)

type ServiceType int

const (
	CluserIP ServiceType = iota
	NodePort
)

type Service struct {
	Type   ServiceType
	Id     string
	IpAddr string
}

func InitKubeServicesChain() error {
	// iptables -t nat -N KUBE-SERVICES
	if err := net.NewIPTablesChain(KUBE_SERVICES_CHAIN); err != nil {
		return err
	}

	// iptables -t nat -A PREROUTING -j KUBE-SERVICES
	if err := net.AppendNewIPTablesRule(fmt.Sprintf("-j %s", KUBE_SERVICES_CHAIN), "PREROUTING"); err != nil {
		return err
	}

	// iptables -t nat -A OUTPUT -j KUBE-SERVICES
	if err := net.AppendNewIPTablesRule(fmt.Sprintf("-j %s", KUBE_SERVICES_CHAIN), "OUTPUT"); err != nil {
		return err
	}

	if err := initMarkChain(); err != nil {
		return err
	}

	// iptables -t nat -A POSTROUTING -m mark --mark 0x4000/0x4000 -j MASQUERADE
	if err := net.AppendNewIPTablesRule("-m mark --mark 0x4000/0x4000 -j MASQUERADE", "POSTROUTING"); err != nil {
		return err
	}

	return nil
}

func initMarkChain() error {
	// iptables -t nat -N KUBE-MARK-MASQ
	if err := net.NewIPTablesChain(KUBE_SERVICE_MARK); err != nil {
		return err
	}

	// iptables -t nat -A KUBE-MARK-MASQ -j MARK --set-xmark 0x4000/0x4000
	if err := net.AppendNewIPTablesRule("-j MARK --set-xmark 0x4000/0x4000", KUBE_SERVICE_MARK); err != nil {
		return err
	}

	return nil
}

func NewClusterIPService(port string, podsCidr string) (*Service, error) {
	const MAX_CHAIN_NAME = 29
	serviceName := pkg.GenerateNewID(CLUSTER_IP_SERVICE_PREFIX)[:MAX_CHAIN_NAME-len(CLUSTER_IP_SERVICE_PREFIX)-1]

	if err := net.NewIPTablesChain(serviceName); err != nil {
		return nil, err
	}

	ipAddr := "172.17.10.10"

	// iptables -t nat -I KUBE-SERVICES 1 ! -s podsCidr -d 1ipAddr -p tcp -m tcp --dport port -j KUBE-MARK-MASQ
	if err := net.InsertNewIPTablesRule(
		fmt.Sprintf("! -s %s -d %s -p tcp -m tcp --dport %s -j %s", podsCidr, ipAddr, port, KUBE_SERVICE_MARK),
		KUBE_SERVICES_CHAIN, 1); err != nil {
		return nil, err
	}

	// iptables -t nat -A KUBE-SERVICES -d clusterIP -p tcp -m tcp --dport port -j ServicerName
	if err := net.AppendNewIPTablesRule(
		fmt.Sprintf("-d %s -p tcp -m tcp --dport %s -j %s", ipAddr, port, serviceName),
		KUBE_SERVICES_CHAIN); err != nil {
		return nil, err
	}

	return &Service{
		Type:   CluserIP,
		Id:     serviceName,
		IpAddr: ipAddr,
	}, nil
}

func AddRouteToClusterIPService(ip string, port string, service string, index int) error {
	podService := fmt.Sprintf(service[:len(service)-3]+"-%d", index)

	if err := net.NewIPTablesChain(podService); err != nil {
		return err
	}

	// iptables -t nat -A podService -s podIp -j KUBE-MARK-MASQ
	if err := net.AppendNewIPTablesRule(fmt.Sprintf("-s %s -j %s", ip, KUBE_SERVICE_MARK), podService); err != nil {
		return err
	}

	// iptables -t nat -A podService -p tcp -m tcp -j DNAT --to-destination route
	if err := net.AppendNewIPTablesRule(fmt.Sprintf("-p tcp -m tcp -j DNAT --to-destination %s", fmt.Sprintf("%s:%s", ip, port)), podService); err != nil {
		return err
	}

	if index == 0 {
		// iptables -t nat -A serviceName -j podService
		return net.AppendNewIPTablesRule(fmt.Sprintf("-j %s", podService), service)
	}

	// iptables -t nat -A service -m statistic --mode nth --every index --packet 0 -j podService
	return net.InsertNewIPTablesRule(fmt.Sprintf("-m statistic --mode nth --every %d --packet 0 -j %s", index+1, podService), service, 1)
}
