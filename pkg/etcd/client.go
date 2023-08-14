package etcd

import (
	"context"
	"os"
	"time"

	"github.com/coreos/etcd/clientv3"
)

var (
	endpoint string
	cli      *clientv3.Client
)

func init() {
	endpoint = os.Getenv("etcd")
}

func Init() error {
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
