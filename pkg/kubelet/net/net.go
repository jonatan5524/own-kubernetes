package net

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/jonatan5524/own-kubernetes/pkg/utils"
)

const ipCommand = "/usr/sbin/ip"

func IsNetDeviceExists(name string) bool {
	_, err := os.Stat(fmt.Sprintf("/sys/class/net/%s", name))

	return !os.IsNotExist(err)
}

func CreateBridge(name string, podCIDR string) error {
	ipAddr := getBridgeIP(podCIDR)

	log.Printf("creating bridge %s %s", name, ipAddr)

	if err := utils.ExecuteCommand(
		fmt.Sprintf("%s link add %s type bridge", ipCommand, name),
		false); err != nil {
		return err
	}

	if err := utils.ExecuteCommand(
		fmt.Sprintf("%s addr add %s dev %s", ipCommand, ipAddr, name),
		false); err != nil {
		return err
	}

	if err := utils.ExecuteCommand(
		fmt.Sprintf("%s link set %s up", ipCommand, name),
		false); err != nil {
		return err
	}

	return nil
}

func getBridgeIP(podCIDR string) string {
	// set bridge IP to 1 in the cidr, for example cidr: 10.1.0.0/16 -> bridgeIP: 10.1.0.1
	podCIDRWithoutMask := podCIDR[:len(podCIDR)-3]

	return utils.ReplaceAtIndex(podCIDRWithoutMask, '1', len(podCIDRWithoutMask)-1)
}

func ConfigurePodNetwork(podID string, bridgeName string, podCIDR string, nsNetPath string) (string, error) {
	podIP, err := getNextAvailableIPAddr(podCIDR)
	if err != nil {
		return "", err
	}

	bridgeIP := getBridgeIP(podCIDR)

	const maxDeviceName = 15
	netID := podID[:maxDeviceName-len("veth")-1]
	if err := createVethPairNamespaces(
		fmt.Sprintf("veth-%s", netID),
		fmt.Sprintf("ceth-%s", netID),
		bridgeName,
		nsNetPath,
		podIP,
		bridgeIP,
	); err != nil {
		return "", err
	}

	return podIP, nil
}

func getNextAvailableIPAddr(cidr string) (string, error) {
	hosts, err := hosts(cidr)
	if err != nil {
		return "", err
	}

	for _, ip := range hosts {
		if err := utils.ExecuteCommand(
			fmt.Sprintf("/usr/bin/ping -c1 -t1 %s", ip),
			true,
		); err != nil {
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

func createVethPairNamespaces(name string, pair string, bridge string, nsNetPath string, ipAddr string, bridgeIPAddr string) error {
	if err := utils.ExecuteCommand(
		fmt.Sprintf("/usr/sbin/ip link add %s type veth peer name %s", name, pair),
		false,
	); err != nil {
		return err
	}

	if err := utils.ExecuteCommand(
		fmt.Sprintf("/usr/sbin/ip link set %s up", name),
		false,
	); err != nil {
		return err
	}

	if err := utils.ExecuteCommand(
		fmt.Sprintf("/usr/sbin/ip link set %s netns %s", pair, nsNetPath),
		false,
	); err != nil {
		return err
	}

	if err := utils.ExecuteCommand(
		fmt.Sprintf("/usr/bin/nsenter --net=%s ip link set %s up", nsNetPath, pair),
		false,
	); err != nil {
		return err
	}

	if err := utils.ExecuteCommand(
		fmt.Sprintf("/usr/bin/nsenter --net=%s /usr/sbin/ip addr add %s dev %s", nsNetPath, ipAddr, pair),
		false,
	); err != nil {
		return err
	}

	if err := utils.ExecuteCommand(
		fmt.Sprintf("/usr/sbin/ip link set %s master %s", name, bridge),
		false,
	); err != nil {
		return err
	}

	return nil
}
