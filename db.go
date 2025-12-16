package redis_tool

import (
	"sync/atomic"

	"github.com/zly-app/component/redis"
	"github.com/zly-app/zapp"
	"github.com/zly-app/zapp/core"
	"github.com/zly-app/zapp/handler"
)

var RedisClientName = "redis_tool"

var isManualInit = false
var onceInit int32 // 只能初始化一次

func GetRedis() (redis.UniversalClient, error) {
	return redis.GetClient(RedisClientName)
}

func init() {
	zapp.AddHandler(zapp.AfterInitializeHandler, func(app core.IApp, handlerType handler.HandlerType) {
		if isManualInit {
			return
		}
		ManualInit()
	})
}

// 设置为手动初始化, 必须在 zapp.New 之前调用才有效
func SetManualInit() {
	isManualInit = true
}

func ManualInit() {
	if atomic.AddInt32(&onceInit, 1) == 1 {
		tryInjectCode()
	}
}
