package redis_tool

import (
	"github.com/zly-app/component/redis"
	"github.com/zly-app/zapp"
	"github.com/zly-app/zapp/core"
	"github.com/zly-app/zapp/handler"
)

var RedisClientName = "redis_tool"

func GetRedis() (redis.UniversalClient, error) {
	return redis.GetClient(RedisClientName)
}

func init() {
	zapp.AddHandler(zapp.AfterInitializeHandler, func(app core.IApp, handlerType handler.HandlerType) {
		tryInjectCode()
	})
}
