package redis

import (
	"sync"

	"github.com/go-redis/redis/v8"
)

var (
	ListKey     = "near_messsage_log"
	redisClient *redis.Client
	once        = &sync.Once{}
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
		redisClient = rdb
	})
}

func GetClient() *redis.Client {
	return redisClient
}
