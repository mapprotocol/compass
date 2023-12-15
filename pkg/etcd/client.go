package etcd

import (
	"context"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	cli *clientv3.Client
)

func Init(endpoint string) error {
	if endpoint == "" {
		return nil
	}
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{endpoint},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return err
	}
	cli = client

	return nil
}

func Put(ctx context.Context, key, value string) error {
	if cli == nil {
		return nil
	}
	_, err := cli.Put(ctx, key, value)
	if err != nil {
		return err
	}
	return nil
}

func Delete(ctx context.Context, key string) error {
	if cli == nil {
		return nil
	}
	_, err := cli.Delete(ctx, key)
	if err != nil {
		return err
	}
	return nil
}
