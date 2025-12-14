package redis_tool

import (
	"context"

	"github.com/zly-app/component/redis"
	"github.com/zly-app/zapp/log"
	"github.com/zly-app/zapp/pkg/utils"
	"go.uber.org/zap"
)

// Redis 执行模式
type execMode int

const (
	modeNone execMode = iota
	modeFunction
	modeScript
)

// 操作类型
type opType string

// 脚本/函数元数据
type scriptMeta struct {
	name        string
	functionDef string
	scriptDef   string
}

// 脚本定义
var scriptMetas = map[opType]*scriptMeta{}

func registryScriptMeta(op opType, s *scriptMeta) {
	scriptMetas[op] = s
}

// 全局状态
var (
	execModeGlobal = modeNone

	// 存储脚本 SHA1（仅在 modeScript 时使用）
	scriptSHA1s = make(map[opType]string)
)

func tryInjectCode() {
	ctx := utils.Trace.CtxStart(context.Background(), "tryInjectCode")
	defer utils.Trace.CtxEnd(ctx)

	rdb, err := GetRedis()
	if err != nil {
		log.Error(ctx, "GetRedis failed", zap.Error(err))
		return
	}

	if checkSupportFunction(ctx, rdb) {
		if tryInjectFunctions(ctx, rdb) {
			execModeGlobal = modeFunction
			log.Info(ctx, "Redis functions loaded successfully")
			return
		}
	}

	if tryInjectScripts(ctx, rdb) {
		execModeGlobal = modeScript
		log.Info(ctx, "Redis scripts loaded successfully")
		return
	}

	log.Warn(ctx, "Failed to load Redis functions or scripts; will use EVAL on every call")
}

func checkSupportFunction(ctx context.Context, rdb redis.UniversalClient) bool {
	_, err := rdb.FunctionStats(ctx).Result()
	if err != nil {
		log.Debug(ctx, "redis FUNCTION not supported", zap.Error(err))
		return false
	}
	return true
}

func checkFunctionExists(ctx context.Context, rdb redis.UniversalClient, libName string) (bool, error) {
	res, err := rdb.Do(ctx, "FUNCTION", "LIST", "LIBRARYNAME", libName).Result()
	if err != nil {
		return false, err
	}
	libs, _ := res.([]interface{})
	return len(libs) > 0, nil
}

func tryInjectFunctions(ctx context.Context, rdb redis.UniversalClient) bool {
	for op, meta := range scriptMetas {
		exists, err := checkFunctionExists(ctx, rdb, meta.name)
		if err != nil {
			log.Error(ctx, "Check function existence failed", zap.String("op", string(op)), zap.Error(err))
			return false
		}
		if !exists {
			if err := registerFunction(ctx, rdb, meta.functionDef); err != nil {
				log.Error(ctx, "Register function failed", zap.String("op", string(op)), zap.Error(err))
				return false
			}
		}
	}
	return true
}

func registerFunction(ctx context.Context, rdb redis.UniversalClient, code string) error {
	_, err := rdb.Do(ctx, "FUNCTION", "LOAD", "REPLACE", code).Result()
	return err
}

func tryInjectScripts(ctx context.Context, rdb redis.UniversalClient) bool {
	for op, meta := range scriptMetas {
		sha, err := rdb.ScriptLoad(ctx, meta.scriptDef).Result()
		if err != nil {
			log.Error(ctx, "Script load failed", zap.String("op", string(op)), zap.Error(err))
			return false
		}
		scriptSHA1s[op] = sha
	}
	return true
}

// ========== 统一执行入口 ==========
func evalRedis(ctx context.Context, op opType, keys []string, args ...any) (any, error) {
	rdb, err := GetRedis()
	if err != nil {
		return 0, err
	}

	switch execModeGlobal {
	case modeFunction:
		// FCALL <name> <numkeys> [key [key ...]] [arg [arg ...]]
		cmdArgs := make([]interface{}, 0, 3+len(keys)+len(args))
		cmdArgs = append(cmdArgs, "FCALL", scriptMetas[op].name, "0") // redis Function 不支持key, 所有参数都应该作为 args
		for _, k := range keys {
			cmdArgs = append(cmdArgs, k)
		}
		cmdArgs = append(cmdArgs, args...)
		res, err := rdb.Do(ctx, cmdArgs...).Result()
		return res, err

	case modeScript:
		res, err := rdb.EvalSha(ctx, scriptSHA1s[op], keys, args...).Result()
		return res, err

	default: // fallback to EVAL
		res, err := rdb.Eval(ctx, scriptMetas[op].scriptDef, keys, args...).Result()
		return res, err
	}
}
