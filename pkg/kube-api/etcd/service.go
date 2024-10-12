package etcd

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type EtcdService interface {
	GetResource(string) ([]byte, error)
	GetAllFromResource(string) ([][]byte, error)
	PutResource(string, string) error
	DeleteResource(string) error
	GetWatchChannel(string) (clientv3.WatchChan, func(), error)
}

type EtcdServiceApp struct {
	endpoints string
}

func NewEtcdService(endpoint string) EtcdService {
	return &EtcdServiceApp{
		endpoints: endpoint,
	}
}

func connect() (*clientv3.Client, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{fmt.Sprintf("%s:2379", os.Getenv("ETCD_ENDPOINT"))},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("error connecting to etcd: %v", err)
	}

	return cli, nil
}

func (app *EtcdServiceApp) GetResource(key string) ([]byte, error) {
	cli, err := connect()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	resp, err := cli.Get(context.Background(), key)
	// cancel()
	if err != nil {
		return nil, fmt.Errorf("failed to get: %v", err)
	}

	if len(resp.Kvs) == 0 {
		return nil, fmt.Errorf("key not found for: %s", key)
	}

	return resp.Kvs[0].Value, nil
}

func (app *EtcdServiceApp) GetAllFromResource(key string) ([][]byte, error) {
	cli, err := connect()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	resp, err := cli.Get(context.Background(), key, clientv3.WithPrefix())
	// cancel()
	if err != nil {
		return nil, fmt.Errorf("failed to get: %v", err)
	}

	if len(resp.Kvs) == 0 {
		return nil, fmt.Errorf("key not found for: %s", key)
	}

	var values [][]byte
	for _, key := range resp.Kvs {
		values = append(values, key.Value)
	}

	return values, nil
}

func (app *EtcdServiceApp) PutResource(key string, value string) error {
	cli, err := connect()
	if err != nil {
		return err
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, err = cli.Put(ctx, key, value)
	defer cancel()

	if err != nil {
		return fmt.Errorf("failed to put: %v", err)
	}

	return nil
}

func (app *EtcdServiceApp) DeleteResource(key string) error {
	cli, err := connect()
	if err != nil {
		return err
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, err = cli.Delete(ctx, key)
	defer cancel()

	if err != nil {
		return fmt.Errorf("failed to delete: %v", err)
	}

	return nil
}

func (app *EtcdServiceApp) GetWatchChannel(key string) (clientv3.WatchChan, func(), error) {
	cli, err := connect()
	if err != nil {
		return nil, nil, err
	}

	watchChan := cli.Watch(context.Background(), key, clientv3.WithPrefix())

	closeChan := func() {
		log.Printf("closing watch channel %s", key)

		cli.Close()
	}

	return watchChan, closeChan, nil
}
