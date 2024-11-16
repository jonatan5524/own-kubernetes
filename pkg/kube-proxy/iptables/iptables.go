package iptables

import (
	"fmt"

	"github.com/jonatan5524/own-kubernetes/pkg/utils"
)

const (
	netTable               = "nat"
	kubeServicesChain      = "KUBE-SERVICES"
	kubeServicesMark       = "KUBE-MARK-MASQ"
	clusterIPServicePrefix = "KUBE-SVC"
)

func NewClusterIPService(uuid string, port string, podsCidr string) (string, error) {
	const MAX_CHAIN_NAME = 29
	serviceName := uuid[:MAX_CHAIN_NAME-len(uuid)-1]

	if err := newIPTablesChain(serviceName); err != nil {
		return "", err
	}

	ipAddr := "172.17.10.10"

	// iptables -t nat -I KUBE-SERVICES ! -s podsCidr -d ipAddr -p tcp -m tcp --dport port -j KUBE-MARK-MASQ
	if err := insertNewIPTablesRule(
		fmt.Sprintf("! -s %s -d %s -p tcp -m tcp --dport %s -j %s", podsCidr, ipAddr, port, kubeServicesMark),
		kubeServicesChain,
		1,
	); err != nil {
		return "", err
	}

	// iptables -t nat -A KUBE-SERVICES -d clusterIP -p tcp -m tcp --dport port -j ServicerName
	if err := appendNewIPTablesRule(
		fmt.Sprintf("-d %s -p tcp -m tcp --dport %s -j %s", ipAddr, port, serviceName),
		kubeServicesChain,
	); err != nil {
		return "", err
	}

	return ipAddr, nil
}

func InitKubeServicesChain() error {
	// iptables -t nat -N KUBE-SERVICES
	if err := newIPTablesChain(kubeServicesChain); err != nil {
		return err
	}

	// iptables -t nat -A PREROUTING -j KUBE-SERVICES
	if err := appendNewIPTablesRule(fmt.Sprintf("-j %s", kubeServicesChain), "PREROUTING"); err != nil {
		return err
	}

	// iptables -t nat -A OUTPUT -j KUBE-SERVICES
	if err := appendNewIPTablesRule(fmt.Sprintf("-j %s", kubeServicesChain), "OUTPUT"); err != nil {
		return err
	}

	if err := initMarkChain(); err != nil {
		return err
	}

	// iptables -t nat -A POSTROUTING -m mark --mark 0x4000/0x4000 -j MASQUERADE
	if err := appendNewIPTablesRule("-m mark --mark 0x4000/0x4000 -j MASQUERADE", "POSTROUTING"); err != nil {
		return err
	}

	return nil
}

func initMarkChain() error {
	// iptables -t nat -N KUBE-MARK-MASQ
	if err := newIPTablesChain(kubeServicesMark); err != nil {
		return err
	}

	// iptables -t nat -A KUBE-MARK-MASQ -j MARK --set-xmark 0x4000/0x4000
	if err := appendNewIPTablesRule("-j MARK --set-xmark 0x4000/0x4000", kubeServicesMark); err != nil {
		return err
	}

	return nil
}

func newIPTablesChain(name string) error {
	return utils.ExecuteCommand(fmt.Sprintf("/usr/sbin/iptables -t %s -N %s", netTable, name))
}

func appendNewIPTablesRule(rule string, chain string) error {
	return utils.ExecuteCommand(fmt.Sprintf("/usr/sbin/iptables -t %s -A %s %s", netTable, chain, rule))
}

func insertNewIPTablesRule(rule string, chain string, index int) error {
	return utils.ExecuteCommand(fmt.Sprintf("/usr/sbin/iptables -t %s -I %s %d %s", netTable, chain, index, rule))
}
