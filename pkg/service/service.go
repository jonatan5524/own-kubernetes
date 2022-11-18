package service

import (
	"fmt"

	"github.com/jonatan5524/own-kubernetes/pkg/net"
)

const (
	KUBE_SERVICES_CHAIN = "KUBE-SERVICES"
	KUBE_SERVICE_MARK   = "KUBE-MARK-MASQ"
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
	Port   string
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
