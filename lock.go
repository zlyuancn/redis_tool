package redis_tool

import (
	"context"
	"errors"
	"math/rand"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/zly-app/zapp/log"
	"go.uber.org/zap"
)

var (
	LockIsUsedByAnother = errors.New("redis Lock Is Used By Another") // 锁被其它进程占用
)

// 解锁
type KeyUnlock func() error

// 秘钥ttl更新
type KeyTtlRenew func(ctx context.Context, ttl time.Duration) error

// 加锁, 返回可用于解锁和续期的函数
func AutoLock(ctx context.Context, lockKey string, ttl time.Duration) (unlock KeyUnlock, renew KeyTtlRenew, err error,
) {
	checkCode := strconv.FormatInt(time.Now().Unix(), 32) + strconv.FormatInt(rand.Int63n(1e9), 32) // 生成随机授权码
	rdb, err := GetRedis()
	if err != nil {
		return nil, nil, err
	}
	ok, err := rdb.SetNX(ctx, lockKey, checkCode, ttl).Result()
	if err != nil {
		log.Error(ctx, "AutoLock set lock fail.", zap.String("key", lockKey), zap.Error(err))
		return nil, nil, err
	}
	if !ok {
		err = LockIsUsedByAnother
		log.Error(ctx, "AutoLock set lock fail.", zap.String("key", lockKey), zap.Error(err))
		return nil, nil, err
	}

	oneUnlock := int32(0)
	unlock = func() error {
		// 一次性解锁
		if atomic.AddInt32(&oneUnlock, 1) != 1 {
			return nil
		}

		ok, err := CompareAndDel(ctx, lockKey, checkCode)
		if err != nil {
			log.Error(ctx, "Unlock fail.", zap.String("key", lockKey), zap.Error(err))
			return err
		}
		if !ok {
			err = LockIsUsedByAnother
			log.Error(ctx, "Unlock fail.", zap.String("key", lockKey), zap.Error(err))
			return err
		}
		return nil
	}
	renew = func(ctx context.Context, ttl time.Duration) error {
		ok, err := CompareAndExpire(ctx, lockKey, checkCode, ttl)
		if err != nil {
			log.Error(ctx, "Renew fail.", zap.String("key", lockKey), zap.Error(err))
			return err
		}
		if !ok {
			err := LockIsUsedByAnother
			log.Error(ctx, "Renew fail.", zap.String("key", lockKey), zap.Error(err))
			return err
		}
		return nil
	}
	return unlock, renew, nil
}

// 加锁, 返回授权码, 授权码用于解锁和续期
func Lock(ctx context.Context, lockKey string, lockTime time.Duration) (string, error) {
	checkCode := strconv.FormatInt(time.Now().Unix(), 32) + strconv.FormatInt(rand.Int63n(1e9), 32) // 生成随机授权码
	rdb, err := GetRedis()
	if err != nil {
		return "", err
	}
	ok, err := rdb.SetNX(ctx, lockKey, checkCode, lockTime).Result()
	if err != nil {
		log.Error(ctx, "Lock set lock fail.", zap.String("key", lockKey), zap.Error(err))
		return "", err
	}
	if !ok {
		err = LockIsUsedByAnother
		log.Error(ctx, "Lock set lock fail.", zap.String("key", lockKey), zap.Error(err))
		return "", err
	}

	return checkCode, nil
}

// 解锁
func UnLock(ctx context.Context, lockKey, checkCode string) error {
	ok, err := CompareAndDel(ctx, lockKey, checkCode)
	if err != nil {
		log.Error(ctx, "Unlock fail.", zap.String("key", lockKey), zap.Error(err))
		return err
	}
	if !ok {
		err = LockIsUsedByAnother
		log.Error(ctx, "Unlock fail.", zap.String("key", lockKey), zap.Error(err))
		return err
	}
	return nil
}

// 续期
func RenewLock(ctx context.Context, lockKey, checkCode string, ttl time.Duration) error {
	ok, err := CompareAndExpire(ctx, lockKey, checkCode, ttl)
	if err != nil {
		log.Error(ctx, "RenewLock fail.", zap.String("key", lockKey), zap.Error(err))
		return err
	}
	if !ok {
		err = LockIsUsedByAnother
		log.Error(ctx, "RenewLock fail.", zap.String("key", lockKey), zap.Error(err))
		return err
	}
	return nil
}

// 检查lock授权码, key不存在也会返回err
func CheckLockCheckCode(ctx context.Context, lockKey, checkCode string) error {
	rdb, err := GetRedis()
	if err != nil {
		return err
	}
	v, err := rdb.Get(ctx, lockKey).Result()
	if err != nil {
		log.Error(ctx, "CheckLockCheckCode call Get fail.", zap.String("key", lockKey), zap.Error(err))
		return err
	}
	if checkCode != v {
		err = LockIsUsedByAnother
		log.Error(ctx, "CheckLockCheckCode fail.", zap.String("key", lockKey), zap.Error(err))
		return err
	}
	return nil
}
