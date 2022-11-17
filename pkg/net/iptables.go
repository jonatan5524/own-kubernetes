package net

import (
	"fmt"

	"github.com/jonatan5524/own-kubernetes/pkg"
)

const (
	NAT_TABLE = "nat"
)

func NewIPTablesChain(name string) error {
	return pkg.ExecuteCommand(fmt.Sprintf("/usr/sbin/iptables -t %s -N %s", NAT_TABLE, name), true)
}

func AppendNewIPTablesRule(rule string, chain string) error {
	return pkg.ExecuteCommand(fmt.Sprintf("/usr/sbin/iptables -t %s -A %s %s", NAT_TABLE, chain, rule), true)
}

func InsertNewIPTablesRule(rule string, chain string, index int) error {
	return pkg.ExecuteCommand(fmt.Sprintf("/usr/sbin/iptables -t %s -I %s %d %s", NAT_TABLE, chain, index, rule), true)
}
