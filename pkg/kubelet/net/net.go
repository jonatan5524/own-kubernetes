package net

import (
	"fmt"
	"log"
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
		fmt.Sprintf("%s link add %s type bridge", ipCommand, name)); err != nil {
		return err
	}

	if err := utils.ExecuteCommand(
		fmt.Sprintf("%s addr add %s dev %s", ipCommand, ipAddr, name)); err != nil {
		return err
	}

	if err := utils.ExecuteCommand(
		fmt.Sprintf("%s link set %s up", ipCommand, name)); err != nil {
		return err
	}

	return nil
}

func getBridgeIP(podCIDR string) string {
	// set bridge IP to 1 in the cidr, for example cidr: 10.1.0.0/16 -> bridgeIP: 10.1.0.1
	podCIDRWithoutMask := podCIDR[:len(podCIDR)-3]

	return utils.ReplaceAtIndex(podCIDRWithoutMask, '1', len(podCIDRWithoutMask)-1)
}

func ConfigurePodNetwork(podID string, bridgeName string, podCIDR string, nsNetPath string, hostNetwork bool) (string, error) {
	podIP, err := utils.GetNextAvailableIPAddr(podCIDR)
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
		hostNetwork,
	); err != nil {
		return "", err
	}

	return podIP, nil
}

func createVethPairNamespaces(name string, pair string, bridge string, nsNetPath string, ipAddr string, bridgeIPAddr string, hostNetwork bool) error {
	if err := utils.ExecuteCommand(
		fmt.Sprintf("/usr/sbin/ip link add %s type veth peer name %s", name, pair),
	); err != nil {
		return err
	}

	if err := utils.ExecuteCommand(
		fmt.Sprintf("/usr/sbin/ip link set %s up", name),
	); err != nil {
		return err
	}

	if err := utils.ExecuteCommand(
		fmt.Sprintf("/usr/sbin/ip link set %s netns %s", pair, nsNetPath),
	); err != nil {
		return err
	}

	if err := utils.ExecuteCommand(
		fmt.Sprintf("/usr/bin/nsenter --net=%s ip link set %s up", nsNetPath, pair),
	); err != nil {
		return err
	}

	if err := utils.ExecuteCommand(
		fmt.Sprintf("/usr/bin/nsenter --net=%s /usr/sbin/ip addr add %s/16 dev %s", nsNetPath, ipAddr, pair),
	); err != nil {
		return err
	}

	if err := utils.ExecuteCommand(
		fmt.Sprintf("/usr/sbin/ip link set %s master %s", name, bridge),
	); err != nil {
		return err
	}

	if !hostNetwork {
		if err := utils.ExecuteCommand(
			fmt.Sprintf("/usr/bin/nsenter --net=%s /usr/sbin/ip route add default via %s", nsNetPath, bridgeIPAddr),
		); err != nil {
			return err
		}
	}

	return nil
}
