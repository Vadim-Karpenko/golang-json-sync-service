package utils

import (
	"context"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
)

func InitializeRedis() *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: os.Getenv("REDIS_PASSWORD"), // no password set
		DB:       0,                           // use default DB
	})

	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		log.Fatal("Could not connect to Redis:", err)
	}

	return rdb
}
