package pod

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"github.com/jonatan5524/own-kubernetes/pkg"
	"github.com/jonatan5524/own-kubernetes/pkg/net"
)

func initContainerdConnection() (*containerd.Client, context.Context, error) {
	client, err := containerd.New(SOCKET_PATH, containerd.WithDefaultNamespace(NAMESPACE))
	if err != nil {
		return nil, nil, err
	}

	ctx := namespaces.WithNamespace(context.Background(), NAMESPACE)

	return client, ctx, nil
}

func NewPodAndRun(imageRegistry string, name string) (string, error) {
	pod, err := NewPod(imageRegistry, name)
	if err != nil {
		return "", err
	}

	log.Printf("pod created: %s\n", pod.Id)
	log.Printf("starting pod\n")

	runningPod, err := pod.Run()
	if err != nil {
		return "", err
	}

	log.Printf("setting up pod network\n")
	if err := connectToNetwork(pod.Id, (*runningPod.Task).Pid()); err != nil {
		return "", err
	}

	return pod.Id, nil
}

func NewPod(imageRegistry string, name string) (*Pod, error) {
	client, ctx, err := initContainerdConnection()
	if err != nil {
		return nil, err
	}

	image, err := client.Pull(ctx, imageRegistry, containerd.WithPullUnpack)
	if err != nil {
		return nil, err
	}

	id := pkg.GenerateNewID(name)

	container, err := client.NewContainer(
		ctx,
		id,
		containerd.WithImage(image),
		containerd.WithNewSnapshot(id+"-snapshot", image),
		containerd.WithNewSpec(oci.WithImageConfig(image)),
	)
	if err != nil {
		return nil, err
	}

	return &Pod{
		Id:        id,
		container: &container,
		ctx:       &ctx,
		client:    client,
	}, nil
}

func connectToNetwork(podId string, pid uint32) error {
	netId := podId[:15-len("veth")-1]

	podCIDR, err := generateNewNodePodCIDR()
	if err != nil {
		return err
	}

	// podCIDR: 10.0.2.0/24 -> bridgeIP: 10.0.2.1/24
	bridgeIP := pkg.ReplaceAtIndex(podCIDR, '1', len(podCIDR)-4)

	if !net.IsDeviceExists(BRIDGE_NAME) {
		if err := net.CreateBridge(BRIDGE_NAME, bridgeIP); err != nil {
			return err
		}
	}

	if !net.IsDeviceExists(VXLAN_NAME) {
		if err := net.CreateVXLAN(VXLAN_NAME, NODE_LOCAL_NETWORK_INTERFACE, BRIDGE_NAME); err != nil {
			return err
		}
	}

	podIP, err := net.GetNextAvailableIPAddr(podCIDR)
	if err != nil {
		return err
	}

	if err := net.CreateVethPairNamespaces(
		fmt.Sprintf("veth-%s", netId),
		fmt.Sprintf("ceth-%s", netId),
		BRIDGE_NAME,
		int(pid),
		podIP+podCIDR[len(podCIDR)-3:],
		bridgeIP,
	); err != nil {
		return err
	}

	return nil
}

func generateNewNodePodCIDR() (string, error) {
	localIPAddr, err := net.GetLocalIPAddr(NODE_LOCAL_NETWORK_INTERFACE)
	if err != nil {
		return "", err
	}

	// localIPAddr: 172.18.0.2 -> podCIDR: 10.0.2.0/24
	return strings.ReplaceAll(POD_CIDR, "x", string(localIPAddr[len(localIPAddr)-4])), nil
}

func ListRunningPods() ([]string, error) {
	client, ctx, err := initContainerdConnection()
	if err != nil {
		return nil, err
	}

	runningPods := []string{}

	containers, err := client.Containers(ctx)
	if err != nil {
		return runningPods, err
	}

	for _, container := range containers {
		_, err = container.Task(ctx, cio.Load)

		if err == nil {
			runningPods = append(runningPods, container.ID())
		}
	}

	return runningPods, nil
}

func LogPod(id string) (string, error) {
	logs, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", LOGS_PATH, id))
	log.Println(fmt.Sprintf("%s/%s", LOGS_PATH, id))

	if err != nil {
		return "", err
	}

	return string(logs), nil
}

func KillPod(id string) (string, error) {
	client, ctx, err := initContainerdConnection()
	if err != nil {
		return "", err
	}

	container, err := client.LoadContainer(ctx, id)
	if err != nil {
		return "", err
	}

	task, err := container.Task(ctx, cio.Load)
	if err != nil {
		return "", err
	}

	exitStatusC, err := task.Wait(ctx)
	if err != nil {
		return "", err
	}

	runningPod := RunningPod{
		Task: &task,
		Pod: &Pod{
			client:    client,
			ctx:       &ctx,
			container: &container,
			Id:        id,
		},
		exitStatusC: exitStatusC,
	}

	_, err = runningPod.Kill()
	if err != nil {
		return "", err
	}

	runningPod.Pod.Delete()

	return id, nil
}
