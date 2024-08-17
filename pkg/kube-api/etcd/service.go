package etcd

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	clientv3 "go.etcd.io/etcd/clientv3"
)

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

func GetResource(key string) ([]byte, error) {
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

func PutResource(key string, value string) error {
	cli, err := connect()
	if err != nil {
		return err
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, err = cli.Put(ctx, key, value)
	defer cancel()

	if err != nil {
		log.Fatal("failed to put: ", err)
	}

	return nil
}

func DeleteResource(key string) error {
	cli, err := connect()
	if err != nil {
		return err
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, err = cli.Delete(ctx, key)
	defer cancel()

	if err != nil {
		log.Fatal("failed to delete: ", err)
	}

	return nil
}
