package redis

import (
	"github.com/go-redis/redis/v8"
)

var (
	ListKey     = "near_messsage_log"
	redisClient *redis.Client
)

func init() {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "46.137.199.126:6379",
		Password: "F6U3gV0L6Xwyw1Ko", // 密码
		DB:       0,                  // 数据库
		PoolSize: 20,                 // 连接池大小
	})

	redisClient = rdb
}

func GetClient() *redis.Client {
	return redisClient
}