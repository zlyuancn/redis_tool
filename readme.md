
# Redis工具

这是一个基于Redis实现的工具库，提供了简单易用的锁，原子操作等功能。

# 功能特性

- **分布式锁**: 基于Redis实现的分布式锁，确保在多实例环境下同一时间只有一个实例能获得锁
- **自动续期**: 提供锁续期功能，防止锁因超时而被意外释放，通过校验码验证确保只有加锁者才能续期
- **安全解锁**: 通过校验码验证确保只有加锁者才能解锁
- **原子操作**: 使用Redis Lua脚本确保操作的原子性操作工具
- 自动加载和缓存 Lua 脚本 SHA1 值，提高执行效率

# 依赖

- github.com/zly-app/zapp
- github.com/zly-app/component/redis

该工具库使用名为 `redis_tool` 的 Redis 客户端名称，你需要确保在应用配置中提供了相应的 Redis 连接配置。

# 安装

```bash
go get github.com/zlyuancn/redis_tool
```

---

# 锁

## AutoLock 自动模式加锁（推荐）

自动加锁，返回解锁函数和续期函数。

```go
func AutoLock(ctx context.Context, lockKey string, ttl time.Duration) (unlock KeyUnlock, renew KeyTtlRenew, err error)
```

**参数:**
- `ctx`: 上下文对象
- `lockKey`: 锁的键名
- `ttl`: 锁的过期时间

**返回值:**
- `KeyUnlock`: 解锁函数
- `KeyTtlRenew`: 续期函数
- `error`: 错误信息

**示例:**
```go
unlock, renew, err := redis_tool.AutoLock(ctx, "my_lock", 30*time.Second)
if err != nil {
    if err == redis_tool.LockIsUsedByAnother {
        fmt.Println("获取锁失败，已被其他进程占用")
        return
    }
    // 处理其他错误
}

defer func() {
    _ = unlock() // 自动解锁
}()

// 业务逻辑
// 续期锁
_ = renew(ctx, 30*time.Second)
```

## Lock 加锁

加锁，返回校验码。

```go
func Lock(ctx context.Context, lockKey string, lockTime time.Duration) (string, error)
```

**参数:**
- `ctx`: 上下文对象
- `lockKey`: 锁的键名
- `lockTime`: 锁的过期时间

**返回值:**
- `string`: 校验码（用于解锁和续期）
- `error`: 错误信息

## UnLock 解锁

```go
func UnLock(ctx context.Context, lockKey, checkCode string) error
```

**参数:**
- `ctx`: 上下文对象
- `lockKey`: 锁的键名
- `checkCode`: 校验码

**返回值:**
- `error`: 错误信息

**示例:**
```go
checkCode, err := redis_tool.Lock(ctx, "my_lock", 30*time.Second)
if err != nil {
    if err == redis_tool.LockIsUsedByAnother {
        fmt.Println("获取锁失败锁，已被其他进程占用")
        return
    }
    // 处理错误
}

// 业务逻辑

// 解锁
err = redis_tool.UnLock(ctx, "my_lock", checkCode)
if err != nil {
    if err == redis_tool.LockIsUsedByAnother {
        fmt.Println("锁已被其他进程占用")
    }
    // 处理其他错误
}
```

## RenewLock 锁续期

```go
func RenewLock(ctx context.Context, lockKey, checkCode string, ttl time.Duration) error
```

**参数:**
- `ctx`: 上下文对象
- `lockKey`: 锁的键名
- `checkCode`: 校验码
- `ttl`: 新的过期时间

**返回值:**
- `error`: 错误信息

**示例:**
```go
checkCode, err := redis_tool.Lock(ctx, "my_lock", 10*time.Second)
if err != nil {
    // 处理错误
}

// 业务逻辑可能需要较长时间，续期锁
err = redis_tool.RenewLock(ctx, "my_lock", checkCode, 10*time.Second)
if err != nil {
    if err == redis_tool.LockIsUsedByAnother {
        fmt.Println("无法续期，锁可能已被其他进程占用")
    }
    // 处理其他错误
}
```
## CheckLockCheckCode 检查锁状态

检查锁的校验码是否匹配，如果锁不存在也会返回错误。

```go
func CheckLockCheckCode(ctx context.Context, lockKey, checkCode string) error
```

**参数:**
- `ctx`: 上下文对象
- `lockKey`: 锁的键名
- `checkCode`: 校验码

**返回值:**
- `error`: 错误信息

**示例:**
```go
err = redis_tool.CheckLockCheckCode(ctx, "my_lock", checkCode)
if err != nil {
    if err == redis_tool.LockIsUsedByAnother {
        fmt.Println("锁的校验码已改变，可能已被其他进程占用")
    }
    // 处理其他错误
}
```

## 错误类型

- `LockIsUsedByAnother`: 锁被其它进程占用

## 注意事项

1. 使用`AutoLock`时，建议使用`defer`确保锁被正确释放
2. 续期操作需要在锁过期前完成，否则可能导致续期失败
3. 如果返回`LockIsUsedByAnother`错误，说明锁可能已被其他实例占用

---

# 原子操作

## CompareAndSwap 原子交换

原子交换操作：如果 key 的值等于 v1，则将其设置为 v2。

```go
func CompareAndSwap(ctx context.Context, key, v1, v2 string) (bool, error)
```

**参数:**
- `ctx`: 上下文对象
- `key`: Redis 键名
- `v1`: 期望的当前值
- `v2`: 要设置的新值

**返回值:**
- `bool`: 操作是否成功
- `error`: 错误信息

**示例:**
```go
success, err := redis_tool.CompareAndSwap(ctx, "mykey", "old_value", "new_value")
if err != nil {
    // 处理错误
}
if success {
    fmt.Println("交换成功")
} else {
    fmt.Println("交换失败，值已被修改")
}
```

## CompareAndDel 原子删除

原子删除操作：如果 key 的值等于 value，则删除该键。

```go
func CompareAndDel(ctx context.Context, key, value string) (bool, error)
```

**参数:**
- `ctx`: 上下文对象
- `key`: Redis 键名
- `value`: 期望的当前值

**返回值:**
- `bool`: 操作是否成功（键不存在返回true）
- `error`: 错误信息

**示例:**
```go
success, err := redis_tool.CompareAndDel(ctx, "mykey", "expected_value")
if err != nil {
    // 处理错误
}
if success {
    fmt.Println("删除成功或键不存在")
} else {
    fmt.Println("删除失败，值不匹配")
}
```

## CompareAndExpire 原子续期

原子续期操作：如果 key 的值等于 value，则更新键的过期时间。

```go
func CompareAndExpire(ctx context.Context, key, value string, ttl time.Duration) (bool, error)
```

**参数:**
- `ctx`: 上下文对象
- `key`: Redis 键名
- `value`: 期望的当前值
- `ttl`: 新的过期时间

**返回值:**
- `bool`: 操作是否成功
- `error`: 错误信息

**示例:**
```go
success, err := redis_tool.CompareAndExpire(ctx, "mykey", "current_value", 30*time.Second)
if err != nil {
    // 处理错误
}
if success {
    fmt.Println("续期成功")
} else {
    fmt.Println("续期失败，值不匹配")
}
```
