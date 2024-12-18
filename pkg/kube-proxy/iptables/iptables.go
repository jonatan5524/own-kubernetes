package iptables

import (
	"crypto/sha256"
	"encoding/base32"
	"fmt"

	"github.com/jonatan5524/own-kubernetes/pkg/utils"
)

const (
	netTable                 = "nat"
	preroutingChain          = "PREROUTING"
	postroutingChain         = "POSTROUTING"
	outputChain              = "OUTPUT"
	kubeServicesChain        = "KUBE-SERVICES"
	kubeServicesMark         = "KUBE-MARK-MASQ"
	kubePostrouting          = "KUBE-POSTROUTING"
	clusterIPServicePrefix   = "KUBE-SVC"
	serviceEndpointPrefix    = "KUBE-SEP"
	nodePortServiceChain     = "KUBE-NODEPORT"
	nodePortServiceExtPrefix = "KUBE-EXT"
)

func CheckIfClusterIPServiceExists(namespace string, serviceName string, portName string) bool {
	id := chainHashPrefix(serviceName, namespace, portName)
	serviceNameChain := fmt.Sprintf("%s-%s", clusterIPServicePrefix, id)

	err := utils.ExecuteCommand(
		fmt.Sprintf("/usr/sbin/iptables -t %s -L %s", netTable, serviceNameChain),
	)

	return err == nil
}

func CheckIfNodePortServiceExists(namespace string, serviceName string, portName string) bool {
	id := chainHashPrefix(serviceName, namespace, portName)
	serviceNameChain := fmt.Sprintf("%s-%s", nodePortServiceExtPrefix, id)

	err := utils.ExecuteCommand(
		fmt.Sprintf("/usr/sbin/iptables -t %s -L %s", netTable, serviceNameChain),
	)

	return err == nil
}

func CheckIfClusterIPServiceEndpointExists(namespace string, serviceName string, portName string) bool {
	id := chainHashPrefix(serviceName, namespace, portName)
	serviceNameChain := fmt.Sprintf("%s-%s", serviceEndpointPrefix, id)

	err := utils.ExecuteCommand(
		fmt.Sprintf("/usr/sbin/iptables -t %s -L %s", netTable, serviceNameChain),
	)

	return err == nil
}

func NewNodePortService(namespace string, serviceName string, port int, portName string) error {
	id := chainHashPrefix(serviceName, namespace, portName)
	serviceNameChain := fmt.Sprintf("%s-%s", nodePortServiceExtPrefix, id)

	// iptables -t nat -N KUBE-EXT-id
	if err := newIPTablesChain(serviceNameChain); err != nil {
		return err
	}

	// iptables -A KUBE-NODEPORT -p tcp -m tcp --dport nodePort -m comment --comment "namespaces/podName" -j KUBE-EXT-id
	if err := appendNewIPTablesRule(
		netTable,
		fmt.Sprintf("-p tcp -m tcp --dport %d -j %s", port, serviceNameChain),
		nodePortServiceChain,
		fmt.Sprintf("%s/%s-service:%s", namespace, serviceName, portName),
	); err != nil {
		return err
	}

	// iptables -A KUBE-EXT-id -m comment --comment "masquerade traffic for namespace/podName external destinations" -j KUBE-MARK-MASQ
	if err := appendNewIPTablesRule(
		netTable,
		fmt.Sprintf("-j %s", kubeServicesMark),
		serviceNameChain,
		fmt.Sprintf("masquerade-traffic-for-%s/%s-external-destinations", namespace, serviceName),
	); err != nil {
		return err
	}

	// iptables -A KUBE-EXT-id  -j KUBE-SVC-id
	if err := appendNewIPTablesRule(
		netTable,
		fmt.Sprintf("-j %s", fmt.Sprintf("%s-%s", clusterIPServicePrefix, chainHashPrefix(serviceName, namespace, portName))),
		serviceNameChain,
		fmt.Sprintf("%s/%s-service:%s", namespace, serviceName, portName),
	); err != nil {
		return err
	}

	return nil
}

func NewClusterIPService(clusterIP string, podCIDR string, namespace string, serviceName string, servicePort int, portName string) error {
	id := chainHashPrefix(serviceName, namespace, portName)
	serviceNameChain := fmt.Sprintf("%s-%s", clusterIPServicePrefix, id)

	// iptables -t nat -N KUBE-SVC-id
	if err := newIPTablesChain(serviceNameChain); err != nil {
		return err
	}

	// iptables -t nat -I KUBE-SERVICES 1 -d ipAddr/32 -p tcp -m tcp --dport servicePort -m comment -j KUBE-SVC-id --comment "namesapce/serviceName cluster IP"
	if err := insertNewIPTablesRule(
		netTable,
		fmt.Sprintf("-d %s/32 -p tcp -m tcp --dport %d -j %s", clusterIP, servicePort, serviceNameChain),
		kubeServicesChain,
		1,
		fmt.Sprintf("%s/%s:%s-clusterIP", namespace, serviceName, portName),
	); err != nil {
		return err
	}

	// iptables -t nat -A KUBE-SVC-id ! -s podCIDR/16 -d clusterIP/32 -p tcp -m tcp --dport servicePort -m comment --comment "namespace/serviceName cluster IP" -j KUBE-MARK-MASQ
	if err := appendNewIPTablesRule(
		netTable,
		fmt.Sprintf("! -s %s -d %s/32 -p tcp -m tcp --dport %d -j %s", podCIDR, clusterIP, servicePort, kubeServicesMark),
		serviceNameChain,
		fmt.Sprintf("%s/%s:%s-clusterIP", namespace, serviceName, portName),
	); err != nil {
		return err
	}

	return nil
}

func ClearClusterIPServiceFromEndpoints(serviceName string, namespace string, portName string) error {
	id := chainHashPrefix(serviceName, namespace, portName)
	serviceNameChain := fmt.Sprintf("%s-%s", clusterIPServicePrefix, id)

	// iptables -t nat -D KUBE-SVC-id 2
	if err := deleteIPTablesRule(
		netTable,
		serviceNameChain,
		2,
	); err != nil {
		return err
	}

	return nil
}

func DeleteServiceEndpoint(podName string, namespace string, portName string) error {
	serviceEndpointChain := fmt.Sprintf("%s-%s", serviceEndpointPrefix, chainHashPrefix(podName, namespace, portName))

	return deleteIPTablesChain(netTable, serviceEndpointChain)
}

func CreateEndpointChain(namespace string, serviceName string, podName string, portName string, podIP string, podPort int) error {
	serviceEndpointChain := fmt.Sprintf("%s-%s", serviceEndpointPrefix, chainHashPrefix(podName, namespace, portName))

	// iptables -t nat -N KUBE-SEP-id
	if err := newIPTablesChain(serviceEndpointChain); err != nil {
		return err
	}

	// -A KUBE-SEP-id -s podIP/32 -m comment --comment "namespace/serviceName" -j KUBE-MARK-MASQ
	if err := appendNewIPTablesRule(
		netTable,
		fmt.Sprintf("-s %s/32 -j %s", podIP, kubeServicesMark),
		serviceEndpointChain,
		fmt.Sprintf("%s/%s:%s-clusterIP", namespace, serviceName, portName),
	); err != nil {
		return err
	}

	// -A KUBE-SEP-id -p tcp -m comment --comment "namespace/serviceName" -m tcp -j DNAT --to-destination podIP:podPort
	if err := appendNewIPTablesRule(
		netTable,
		fmt.Sprintf("-p tcp -m tcp -j DNAT --to-destination %s:%d", podIP, podPort),
		serviceEndpointChain,
		fmt.Sprintf("%s/%s:%s-clusterIP", namespace, serviceName, portName),
	); err != nil {
		return err
	}

	return nil
}

func AddEndpointToServiceChain(namespace string, serviceName string, podName string, portName string, podIP string, podPort int, probability float32) error {
	serviceNameChain := fmt.Sprintf("%s-%s", clusterIPServicePrefix, chainHashPrefix(serviceName, namespace, portName))
	serviceEndpointChain := fmt.Sprintf("%s-%s", serviceEndpointPrefix, chainHashPrefix(podName, namespace, portName))

	// iptables -A KUBE-SVC-id -m comment --comment "namespace/serviceName->podIP:podPort" -m statistic --mode random --probability probability -j KUBE-SEP-id
	rule := ""
	if probability != 0 {
		rule = fmt.Sprintf("-m statistic --mode random --probability %f -j %s", probability, serviceEndpointChain)
	} else {
		rule = fmt.Sprintf("-j %s", serviceEndpointChain)
	}
	if err := insertNewIPTablesRule(
		netTable,
		rule,
		serviceNameChain,
		2,
		fmt.Sprintf("%s/%s->%s:%d", namespace, serviceName, podIP, podPort),
	); err != nil {
		return err
	}

	return nil
}

// Taken from kuberenetes source code:
// https://github.com/kubernetes/kubernetes/blob/da215bf06a3b8ac3da4e0adb110dc5acc7f61fe1/pkg/proxy/iptables/proxier.go#L707-L708
func chainHashPrefix(serviceName string, namespace string, portName string) string {
	hash := sha256.Sum256([]byte(namespace + serviceName + portName))
	encoded := base32.StdEncoding.EncodeToString(hash[:])
	return encoded[:16]
}

func InitKubeServicesChain() error {
	// iptables -t nat -N KUBE-SERVICES
	if err := newIPTablesChain(kubeServicesChain); err != nil {
		return err
	}

	// iptables -t nat -A PREROUTING -j KUBE-SERVICES -m comment --comment 'kubernetes services'
	if err := appendNewIPTablesRule(
		netTable,
		fmt.Sprintf("-j %s", kubeServicesChain),
		preroutingChain,
		"kubernetes-services",
	); err != nil {
		return err
	}

	// iptables -t nat -A OUTPUT -j KUBE-SERVICES
	if err := appendNewIPTablesRule(
		netTable,
		fmt.Sprintf("-j %s", kubeServicesChain),
		outputChain,
		"kubernetes-services",
	); err != nil {
		return err
	}

	if err := initMarkChain(); err != nil {
		return err
	}

	if err := initKubePostRouting(); err != nil {
		return err
	}

	return nil
}

func InitNodePortChain() error {
	// iptables -t nat -N KUBE-NODEPORTS
	if err := newIPTablesChain(nodePortServiceChain); err != nil {
		return err
	}

	// iptables -t nat -A KUBE-SERVICES -m addrtype --dst-type LOCAL -j KUBE-NODEPORTS
	if err := appendNewIPTablesRule(
		netTable,
		fmt.Sprintf("-m addrtype --dst-type LOCAL -j %s",
			nodePortServiceChain),
		kubeServicesChain,
		"kubernetes-service-nodeports",
	); err != nil {
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
	if err := appendNewIPTablesRule(
		netTable,
		"-j MARK --set-xmark 0x4000/0x4000",
		kubeServicesMark,
		"kubernetes-service-mark",
	); err != nil {
		return err
	}

	return nil
}

func initKubePostRouting() error {
	// iptables -t nat -N KUBE-POSTROUTING
	if err := newIPTablesChain(kubePostrouting); err != nil {
		return err
	}

	// iptables -t nat -A POSTROUTING -j KUBE-POSTROUTING -m comment --comment "kuberenetes-postrouting-rules"
	if err := appendNewIPTablesRule(
		netTable,
		fmt.Sprintf("-j %s", kubePostrouting),
		postroutingChain,
		"kubernetes-postrouting-rules"); err != nil {
		return err
	}

	// iptables -t nat -A KUBE-POSTROUTING -m mark ! --mark 0x4000/0x4000 -j RETURN
	if err := appendNewIPTablesRule(
		netTable,
		"-m mark ! --mark 0x4000/0x4000 -j RETURN",
		kubePostrouting,
		""); err != nil {
		return err
	}

	// iptables -t nat -A KUBE-POSTROUTING -j MARK --set-xmark 0x4000/0x0
	if err := appendNewIPTablesRule(
		netTable,
		"-j MARK --set-xmark 0x4000/0x0",
		kubePostrouting,
		""); err != nil {
		return err
	}

	// iptables -t nat -A KUBE-POSTROUTING -m comment --comment "kubernetes-service-traffic-requiring-SNAT" -j MASQUERADE --random-fully
	if err := appendNewIPTablesRule(
		netTable,
		"-j MASQUERADE --random-fully",
		kubePostrouting,
		"kubernetes-service-traffic-requiring-SNAT"); err != nil {
		return err
	}

	return nil
}

func newIPTablesChain(name string) error {
	return utils.ExecuteCommand(fmt.Sprintf("/usr/sbin/iptables -t %s -N %s", netTable, name))
}

func appendNewIPTablesRule(table string, rule string, chain string, comment string) error {
	return utils.ExecuteCommand(
		fmt.Sprintf("/usr/sbin/iptables -t %s -A %s -m comment --comment \"%s\" %s",
			table,
			chain,
			comment,
			rule,
		),
	)
}

func insertNewIPTablesRule(table string, rule string, chain string, index int, comment string) error {
	return utils.ExecuteCommand(
		fmt.Sprintf("/usr/sbin/iptables -t %s -I %s %d -m comment --comment \"%s\" %s",
			table,
			chain,
			index,
			comment,
			rule,
		),
	)
}

func deleteIPTablesRule(table string, chain string, index int) error {
	return utils.ExecuteCommand(
		fmt.Sprintf("/usr/sbin/iptables -t %s -D %s %d",
			table,
			chain,
			index,
		),
	)
}

func deleteIPTablesChain(table string, chain string) error {
	err := utils.ExecuteCommand(
		fmt.Sprintf("/usr/sbin/iptables -t %s -F %s",
			table,
			chain,
		),
	)
	if err != nil {
		return err
	}

	return utils.ExecuteCommand(
		fmt.Sprintf("/usr/sbin/iptables -t %s -X %s",
			table,
			chain,
		),
	)
}
