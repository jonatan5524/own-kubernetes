package containerd

import (
	"context"
	"fmt"
	"log"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"github.com/google/uuid"
	"github.com/opencontainers/runtime-spec/specs-go"

	kubeapi_rest "github.com/jonatan5524/own-kubernetes/pkg/kube-api/rest"
	"github.com/jonatan5524/own-kubernetes/pkg/utils"
)

const (
	containerdSocketPath = "/run/containerd/containerd.sock"
	defaultNamespace     = "own-kube"
	defaultDNSConfig     = "nameserver 8.8.8.8\nnameserver 8.8.4.4\n"
	defaultEtcHosts      = "127.0.0.1 localhost\n" // TODO: add container ip and container name
)

type CreateContainerSpec struct {
	LogLocation          string
	ResolvConfLocation   string
	HostnameLocation     string
	EtcHostsLocation     string
	NetworkNamespacePath string
	IPCNamespacePath     string
	HostNetwork          bool
	ContainerID          string
}

func containerdConnection() (*containerd.Client, context.Context, error) {
	client, err := containerd.New(containerdSocketPath)
	if err != nil {
		return nil, nil, err
	}

	ctx := namespaces.WithNamespace(context.Background(), defaultNamespace)

	return client, ctx, nil
}

func CreateContainer(
	container *kubeapi_rest.Container,
	createContainerSpec CreateContainerSpec,
) (string, error) {
	log.Printf("creating container %s with containerd", container.Name)

	client, ctx, err := containerdConnection()
	if err != nil {
		return "", err
	}
	defer client.Close()

	log.Printf("pulling image %s ", container.Image)
	imageRef, err := client.Pull(ctx, container.Image, containerd.WithPullUnpack)
	if err != nil {
		return "", err
	}

	containerSpec, err := buildContainerSpec(
		imageRef,
		container,
		createContainerSpec,
	)
	if err != nil {
		return "", err
	}

	if createContainerSpec.ContainerID == "" {
		createContainerSpec.ContainerID = uuid.NewString()
	}

	containerRef, err := client.NewContainer(
		ctx,
		createContainerSpec.ContainerID,
		containerd.WithNewSnapshot(container.Name+"-snapshot", imageRef),
		containerd.WithNewSpec(containerSpec...),
	)
	if err != nil {
		return "", err
	}

	err = startContainer(ctx, containerRef, createContainerSpec.LogLocation)
	if err != nil {
		return "", err
	}

	return containerRef.ID(), nil
}

func convertEnvToStringSlice(container *kubeapi_rest.Container) []string {
	env := make([]string, len(container.Env))

	for index, envStruct := range container.Env {
		env[index] = fmt.Sprintf("%s=%s", envStruct.Name, envStruct.Value)
	}

	return env
}

func startContainer(ctx context.Context, container containerd.Container, logLocation string) error {
	log.Printf("starting container %s", container.ID())

	task, err := container.NewTask(ctx, cio.LogFile(logLocation))
	if err != nil {
		return err
	}

	return task.Start(ctx)
}

func getDefaultContainerMounts(container *kubeapi_rest.Container, resolvConfLocation string, hostnameLocation string, etcHostsLocation string) ([]specs.Mount, error) {
	var mounts []specs.Mount

	// TODO: currently public dns server and not coredns
	if err := utils.CreateAndWriteToFile(resolvConfLocation, defaultDNSConfig, 0o644); err != nil {
		return mounts, err
	}
	mounts = append(mounts, specs.Mount{
		Source:      resolvConfLocation,
		Destination: "/etc/resolv.conf",
		Type:        "bind",
		Options:     []string{"rbind", "rw"},
	})

	if err := utils.CreateAndWriteToFile(hostnameLocation, container.Name, 0o644); err != nil {
		return mounts, err
	}
	mounts = append(mounts, specs.Mount{
		Source:      hostnameLocation,
		Destination: "/etc/hostname",
		Type:        "bind",
		Options:     []string{"rbind", "rw"},
	})

	if err := utils.CreateAndWriteToFile(etcHostsLocation, defaultEtcHosts, 0o644); err != nil {
		return mounts, err
	}
	mounts = append(mounts, specs.Mount{
		Source:      etcHostsLocation,
		Destination: "/etc/hosts",
		Type:        "bind",
		Options:     []string{"rbind", "rw"},
	})

	return mounts, nil
}

func buildContainerSpec(
	imageRef oci.Image,
	container *kubeapi_rest.Container,
	createContainerSpec CreateContainerSpec,
) ([]oci.SpecOpts, error) {
	specsOpts := []oci.SpecOpts{}

	mounts, err := getDefaultContainerMounts(
		container,
		createContainerSpec.ResolvConfLocation,
		createContainerSpec.HostnameLocation,
		createContainerSpec.EtcHostsLocation,
	)
	if err != nil {
		return specsOpts, err
	}

	specsOpts = append(specsOpts,
		oci.WithImageConfig(imageRef),
		oci.WithEnv(convertEnvToStringSlice(container)),
		oci.WithMounts(mounts),
	)

	if createContainerSpec.NetworkNamespacePath != "" {
		specsOpts = append(specsOpts,
			oci.WithLinuxNamespace(specs.LinuxNamespace{
				Type: specs.NetworkNamespace,
				Path: createContainerSpec.NetworkNamespacePath,
			}),
		)
	}

	if createContainerSpec.IPCNamespacePath != "" {
		specsOpts = append(specsOpts,
			oci.WithLinuxNamespace(specs.LinuxNamespace{
				Type: specs.IPCNamespace,
				Path: createContainerSpec.IPCNamespacePath,
			}),
		)
	}

	if len(container.Command) != 0 {
		specsOpts = append(specsOpts, oci.WithProcessArgs(append(container.Command, container.Args...)...))
	}

	if createContainerSpec.HostNetwork {
		specsOpts = append(specsOpts, oci.WithHostNamespace(specs.NetworkNamespace))
	}

	return specsOpts, nil
}

func GetContainerPID(containerID string) (uint32, error) {
	currentContainerTask, err := getContainerCurrentTask(containerID)
	if err != nil {
		return 0, err
	}

	return currentContainerTask.Pid(), err
}

func getContainerCurrentTask(containerID string) (containerd.Task, error) {
	log.Printf("get current container %s task", containerID)

	client, ctx, err := containerdConnection()
	if err != nil {
		return nil, err
	}
	defer client.Close()

	containerRef, err := client.LoadContainer(ctx, containerID)
	if err != nil {
		return nil, err
	}

	return containerRef.Task(ctx, cio.Load)
}
