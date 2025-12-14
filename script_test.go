package redis_tool

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/zly-app/component/redis"
	"github.com/zly-app/zapp"
	"github.com/zly-app/zapp/config"
	"github.com/zly-app/zapp/core"
)

func TestMain(m *testing.M) {
	app := zapp.NewApp("test",
		zapp.WithConfigOption(
			config.WithConfig(&core.Config{
				Components: map[string]map[string]interface{}{
					"redis": {
						"redis_tool": redis.RedisConfig{
							Address:  "localhost:6379",
							Password: "",
						},
					},
				},
			}),
		),
	)
	defer app.Exit()

	m.Run()
}

func TestCompareAndSwap(t *testing.T) {
	ctx := context.Background()
	key := "test_cas_key"

	// 清理测试数据
	rdb, _ := GetRedis()
	rdb.Del(ctx, key)

	// 测试场景1: key不存在，设置初始值
	initialValue := "initial_value"
	newValue := "new_value"

	// 先设置一个初始值
	rdb.Set(ctx, key, initialValue, 0)

	// 测试CAS操作：期望值匹配，应成功
	success, err := CompareAndSwap(ctx, key, initialValue, newValue)
	assert.NoError(t, err)
	assert.True(t, success, "当期望值与实际值相匹配时，CAS 应该能够成功。")

	// 验证值已被更改
	currentValue, err := rdb.Get(ctx, key).Result()
	assert.NoError(t, err)
	assert.Equal(t, newValue, currentValue, "成功执行 CAS 操作后，应更新值。")

	// 再次尝试使用旧值进行CAS，应失败
	success, err = CompareAndSwap(ctx, key, initialValue, "another_value")
	assert.NoError(t, err)
	assert.False(t, success, "当预期值与实际值不一致时，CAS 应该会失败。")

	// 验证值未被更改
	currentValue, err = rdb.Get(ctx, key).Result()
	assert.NoError(t, err)
	assert.Equal(t, newValue, currentValue, "在进行 CAS 失败后，key 应保持不变。")

	// 清理
	rdb.Del(ctx, key)
}

func TestCompareAndDel(t *testing.T) {
	ctx := context.Background()
	key := "test_cad_key"

	// 清理测试数据
	rdb, _ := GetRedis()
	rdb.Del(ctx, key)

	// 设置测试值
	testValue := "test_value"
	rdb.Set(ctx, key, testValue, 0)

	// 测试CAD操作：期望值匹配，应成功删除
	success, err := CompareAndDel(ctx, key, testValue)
	assert.NoError(t, err)
	assert.True(t, success, "当预期值与实际值相符时，CAD 就会成功。")

	// 验证key已被删除
	exists, err := rdb.Exists(ctx, key).Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(0), exists, "成功完成 CAD 操作后，应删除 key。")

	// 再次尝试对已删除的key进行CAD，由于key不存在也应成功
	success, err = CompareAndDel(ctx, key, testValue)
	assert.NoError(t, err)
	assert.True(t, success, "当关键参数不存在时，CAD 应该能够正常运行。")

	// 设置新值进行测试
	rdb.Set(ctx, key, "other_value", 0)
	success, err = CompareAndDel(ctx, key, testValue)
	assert.NoError(t, err)
	assert.False(t, success, "当预期值与实际值不一致时，CAD 应该停止运行。")

	// 验证key仍然存在且值未变
	exists, err = rdb.Exists(ctx, key).Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), exists, "在 CAD 失败的情况下，key 仍应存在。")

	currentValue, err := rdb.Get(ctx, key).Result()
	assert.NoError(t, err)
	assert.Equal(t, "other_value", currentValue, "在 CAD 失败后，key 应保持不变。")

	// 清理
	rdb.Del(ctx, key)
}

func TestCompareAndExpire(t *testing.T) {
	ctx := context.Background()
	key := "test_cae_key"

	// 清理测试数据
	rdb, _ := GetRedis()
	rdb.Del(ctx, key)

	// 设置测试值
	testValue := "test_value"
	ttl := 5 * time.Second
	rdb.Set(ctx, key, testValue, ttl)

	// 测试CAE操作：期望值匹配，应成功续期
	success, err := CompareAndExpire(ctx, key, testValue, 10*time.Second)
	assert.NoError(t, err)
	assert.True(t, success, "当期望值与实际值相匹配时，CAE 就会成功。")

	// 验证TTL是否被更新
	ttlResult, err := rdb.TTL(ctx, key).Result()
	assert.NoError(t, err)
	assert.True(t, ttlResult > 5*time.Second, "在完成 CAE 测试后，应延长 TTL（时间限制）设置。")

	// 使用错误的期望值尝试CAE，应失败
	success, err = CompareAndExpire(ctx, key, "wrong_value", 15*time.Second)
	assert.NoError(t, err)
	assert.False(t, success, "当预期值与实际值不一致时，CAE 应该会失效。")

	// TTL不应被改变
	ttlResultAfterFailed, err := rdb.TTL(ctx, key).Result()
	assert.NoError(t, err)
	// 由于时间流逝，这里只是验证它没有变成15秒
	assert.True(t, ttlResultAfterFailed <= 10*time.Second, "在进行过 CAE 测试失败后，不应更改 TTL 值。")

	// 清理
	rdb.Del(ctx, key)
}
