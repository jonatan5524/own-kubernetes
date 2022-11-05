package net

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"

	"github.com/jonatan5524/own-kubernetes/pkg"
)

func CreateBridge(name string, ipAddr string) error {
	if err := pkg.ExecuteCommand(fmt.Sprintf("ip link add %s type bridge", name)); err != nil {
		return err
	}

	if err := pkg.ExecuteCommand(fmt.Sprintf("ip addr add %s dev %s", ipAddr, name)); err != nil {
		return err
	}

	if err := pkg.ExecuteCommand(fmt.Sprintf("ip link set %s up", name)); err != nil {
		return err
	}

	return nil
}

func CreateVethPairNamespaces(name string, pair string, bridge string, namespacePID int, ipAddr string, bridgeIpAddr string) error {
	if err := pkg.ExecuteCommand(fmt.Sprintf("ip link add %s type veth peer name %s", name, pair)); err != nil {
		return err
	}

	if err := pkg.ExecuteCommand(fmt.Sprintf("ip link set %s up", name)); err != nil {
		return err
	}

	if err := pkg.ExecuteCommand(fmt.Sprintf("ip link set %s netns /proc/%d/ns/net", pair, namespacePID)); err != nil {
		return err
	}

	if err := pkg.ExecuteCommand(fmt.Sprintf("nsenter --net=/proc/%d/ns/net ip link set %s up", namespacePID, pair)); err != nil {
		return err
	}

	if err := pkg.ExecuteCommand(fmt.Sprintf("nsenter --net=/proc/%d/ns/net ip addr add %s dev %s", namespacePID, ipAddr, pair)); err != nil {
		return err
	}

	if err := pkg.ExecuteCommand(fmt.Sprintf("ip link set %s master %s", name, bridge)); err != nil {
		return err
	}

	if err := pkg.ExecuteCommand(fmt.Sprintf("nsenter --net=/proc/%d/ns/net /usr/sbin/ip route add default via %s", namespacePID, bridgeIpAddr)); err != nil {
		return err
	}

	return nil
}

func CreateVXLAN(name string, nodeInterface string, bridgeName string) error {
	const (
		ID    = "10"
		GROUP = "239.1.1.1"
	)

	if err := pkg.ExecuteCommand(fmt.Sprintf("ip link add %s type vxlan id %s group %s dstport 0 dev %s", name, ID, GROUP, nodeInterface)); err != nil {
		return err
	}

	if err := pkg.ExecuteCommand(fmt.Sprintf("ip link set %s master %s", name, bridgeName)); err != nil {
		return err
	}

	if err := pkg.ExecuteCommand(fmt.Sprintf("ip link set %s up", name)); err != nil {
		return err
	}

	return nil
}

func IsDeviceExists(name string) bool {
	_, err := os.Stat(fmt.Sprintf("/sys/class/net/%s", name))

	return !os.IsNotExist(err)
}

func GetLocalIPAddr(interfaceName string) (addr string, err error) {
	ief, err := net.InterfaceByName(interfaceName)

	if err != nil {
		return "", err
	}

	addrs, err := ief.Addrs()
	if err != nil {
		return "", err
	}

	return addrs[0].String(), nil
}

func GetNextAvailableIPAddr(cidr string) (string, error) {
	hosts, err := hosts(cidr)
	if err != nil {
		return "", err
	}

	for _, ip := range hosts {
		out, _ := exec.Command("ping", "-c1", "-t1", ip).Output()

		if strings.Contains(string(out), "Destination Host Unreachable") {
			return ip, nil
		}
	}

	return "", fmt.Errorf("no available ip have found in cidr: %s", cidr)
}

func hosts(cidr string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	inc := func(ip net.IP) {
		for j := len(ip) - 1; j >= 0; j-- {
			ip[j]++
			if ip[j] > 0 {
				break
			}
		}
	}

	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}

	// remove network address and broadcast address
	return ips[1 : len(ips)-1], nil
}
