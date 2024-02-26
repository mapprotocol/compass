package redis

import (
	"sync"

	"github.com/go-redis/redis/v8"
)

var (
	ListKey    = "near_messsage_log"
	nearClient *redis.Client
	once       = &sync.Once{}
)

func Init(url string) {
	if url == "" {
		panic("messenger redisUrl is empty")
	}
	once.Do(func() {
		opt, err := redis.ParseURL(url)
		if err != nil {
			panic(err)
		}
		rdb := redis.NewClient(opt)
		nearClient = rdb
	})
}

func GetClient() *redis.Client {
	return nearClient
}

func New(url string) (*redis.Client, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	rdb := redis.NewClient(opt)
	return rdb, nil
}
