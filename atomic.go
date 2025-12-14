package redis_tool

import (
	"context"
	"time"

	"github.com/spf13/cast"
)

const (
	opCAS opType = "CAS"
	opCAD opType = "CAD"
	opCAE opType = "CAE"
)

func init() {
	registryScriptMeta(opCAS, &scriptMeta{
		name:        "redis_tool_CAS",
		functionDef: redisFunction_CAS,
		scriptDef:   redisScript_CAS,
	})
	registryScriptMeta(opCAD, &scriptMeta{
		name:        "redis_tool_CAD",
		functionDef: redisFunction_CAD,
		scriptDef:   redisScript_CAD,
	})
	registryScriptMeta(opCAE, &scriptMeta{
		name:        "redis_tool_CAE",
		functionDef: redisFunction_CAE,
		scriptDef:   redisScript_CAE,
	})
}

// ========== Lua 定义 ==========
const (
	// 原子交换, 如果key的值等于v1, 则设为v2; ARGV=[key, v1, v2]; 成功返回1
	redisFunction_CAS = `#!lua name=redis_tool_CAS
local function redis_tool_CAS(keys, args)
    local key = args[1]
    local expected = args[2]
    local new_value = args[3]

    local current_value = redis.pcall('GET', key)
    if current_value == expected then
        redis.pcall('SET', key, new_value)
        return 1
    else
        return 0
    end
end
redis.register_function{function_name='redis_tool_CAS', callback=redis_tool_CAS, description='Atomic swap: If the value of key is equal to v1, then set it to v2; ARGV=[key, v1, v2]; Success returned 1'}`

	// 原子删除, 如果key的值等于value则删除, 如果删除成功或者key不存在则返回1; ARGV=[key, value]
	redisFunction_CAD = `#!lua name=redis_tool_CAD
local function redis_tool_CAD(keys, args)
    local key = args[1]
    local expected = args[2]

    local current_value = redis.pcall('GET', key)
    if current_value == expected then
        redis.pcall('DEL', key)
        return 1
    elseif current_value == false then
        return 1
    else
        return 0
    end
end
redis.register_function{function_name='redis_tool_CAD', callback=redis_tool_CAD, description='Atomic deletion: If the value of the key is equal to the specified value, it will be deleted. If the deletion is successful or the key does not exist, it will return 1; ARGV=[key, value]'}`

	// 原子续期, 如果key的值等于value则续期, 续期成功返回1; ARGV=[key, value, ttl(秒)]
	redisFunction_CAE = `#!lua name=redis_tool_CAE
local function redis_tool_CAE(keys, args)
    local key = args[1]
    local expected = args[2]
    local ttl = tonumber(args[3])

    local current_value = redis.pcall('GET', key)
    if current_value == expected then
        redis.pcall('EXPIRE', key, ttl)
        return 1
    else
        return 0
    end
end
redis.register_function{function_name='redis_tool_CAE', callback=redis_tool_CAE, description='Atomic renewal: If the value of the key is equal to the value specified, the renewal will be performed. If the renewal is successful, it will return 1; ARGV = [key, value, ttl (in seconds)]'}`
)

const (
	// 原子交换, 如果key的值等于v1, 则设为v2; KEYS=[key] ARGV=[v1, v2]; 成功返回1
	redisScript_CAS = `
local v = redis.pcall("get", KEYS[1])
if (v == ARGV[1]) then
    redis.pcall("set", KEYS[1], ARGV[2])
    return 1
end
return 0
`

	// 原子删除, 如果key的值等于value则删除, 如果删除成功或者key不存在则返回1; KEYS=[key] ARGV=[value]
	redisScript_CAD = `
local v = redis.pcall("get", KEYS[1])
if (v == ARGV[1]) then
    redis.pcall("del", KEYS[1])
    return 1
end
if (v == false) then
    return 1
end
return 0
`

	// 原子续期, 如果key的值等于value则续期, 续期成功返回1; KEYS=[key] ARGV=[value, ttl(秒)]
	redisScript_CAE = `
local v = redis.pcall("get", KEYS[1])
if (v == ARGV[1]) then
    redis.pcall("expire", KEYS[1], tonumber(ARGV[2]))
    return 1
end
return 0
`
)

func CompareAndSwap(ctx context.Context, key, v1, v2 string) (bool, error) {
	res, err := evalRedis(ctx, opCAS, []string{key}, v1, v2)
	return cast.ToInt(res) == 1, err
}

func CompareAndDel(ctx context.Context, key, value string) (bool, error) {
	res, err := evalRedis(ctx, opCAD, []string{key}, value)
	return cast.ToInt(res) == 1, err
}

func CompareAndExpire(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	res, err := evalRedis(ctx, opCAE, []string{key}, value, int64(ttl.Seconds()))
	return cast.ToInt(res) == 1, err
}
