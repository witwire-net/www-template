package valkey

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"www-template/packages/backend/internal/platform/observability"
)

const valkeyUnknownTarget = "unknown"

// NewObservedClient は Valkey client に安全な command 観測 hook を登録して返す。
// Product/Admin の store はこの関数経由で client を生成し、直接 redis.NewClient を呼ばない。
//
// 引数:
//   - options: redis.ParseURL で構築済みの接続 option。nil は呼び出し側の構成不備として扱う。
//   - surface: Product/Admin などの論理 surface 名。ログには raw key ではなく namespace 推定にだけ使う。
//
// 戻り値:
//   - *redis.Client: command/pipeline hook 登録済みの Redis 互換 client。
func NewObservedClient(options *redis.Options, surface string) *redis.Client {
	// Step 1: Redis client を生成し、接続・timeout 等の責務は呼び出し元が設定した options に委譲する。
	client := redis.NewClient(options)

	// Step 2: Product/Admin surface 名を hook に渡し、key が空の場合でも target に安全な surface hint を残せるようにする。
	client.AddHook(observationHook{surface: strings.TrimSpace(surface)})
	return client
}

type observationHook struct {
	surface string
}

func (h observationHook) DialHook(next redis.DialHook) redis.DialHook {
	// Step 1: command I/O ではない dial は Redis client の既定動作へ委譲し、hook の責務を command 観測に限定する。
	return next
}

func (h observationHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		// Step 1: 単一 command の実行時間を測り、next 実行後に command metadata だけを記録する。
		startedAt := time.Now()
		err := next(ctx, cmd)
		h.observeCommand(ctx, startedAt, cmd, err)
		return err
	}
}

func (h observationHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		// Step 1: pipeline 全体の開始時刻を共有し、個々の command 結果を同じ I/O batch の一部として記録する。
		startedAt := time.Now()
		err := next(ctx, cmds)
		for _, cmd := range cmds {
			// Step 2: command ごとの Err を優先し、pipeline 全体 error しか無い場合はそれを各 command に反映する。
			commandErr := cmd.Err()
			if commandErr == nil {
				commandErr = err
			}
			h.observeCommand(ctx, startedAt, cmd, commandErr)
		}
		return err
	}
}

func (h observationHook) observeCommand(ctx context.Context, startedAt time.Time, cmd redis.Cmder, err error) {
	// Step 1: command 名と target pattern を raw args から安全に抽出し、値そのものは一切属性化しない。
	commandName := safeValkeyCommandName(cmd)
	args := cmd.Args()
	target := safeValkeyCommandTarget(commandName, args, h.surface)

	// Step 2: error を stable class へ分類し、Redis Nil は key/value 非存在として not_found に寄せる。
	status, errorClass := valkeyObservationStatus(err)
	requestBytes := int64(valkeyRequestArgumentBytes(commandName, args))
	responseBytes := int64(valkeyResponseBytes(cmd))
	resultCount := int64(valkeyResultCount(cmd))

	// Step 3: raw key/value/script を出さず、command metadata と安全な size/count だけを SigNoz へ渡す。
	observability.ObserveDatastoreOperationCompleted(ctx, observability.DatastoreOperationCompleted{
		System:        observability.DatastoreSystemValkey,
		Operation:     commandName,
		Target:        target,
		Status:        status,
		Duration:      time.Since(startedAt),
		ResultCount:   optionalPositiveInt64(resultCount),
		RequestBytes:  optionalPositiveInt64(requestBytes),
		ResponseBytes: optionalPositiveInt64(responseBytes),
		ResultClass:   valkeyResultClass(cmd, errorClass),
		ErrorClass:    errorClass,
	})
}

func valkeyObservationStatus(err error) (observability.DatastoreOperationStatus, observability.DatastoreErrorClass) {
	// Step 1: 成功時は error_class=none として成功/失敗を同じ属性で検索できるようにする。
	if err == nil {
		return observability.DatastoreOperationStatusOK, observability.DatastoreErrorClassNone
	}

	// Step 2: redis.Nil は key/value 不在であり、接続障害とは別の not_found として分類する。
	if errors.Is(err, redis.Nil) {
		return observability.DatastoreOperationStatusNotFound, observability.DatastoreErrorClassNotFound
	}

	// Step 3: context cancellation は caller 起因の中断として DB/Valkey 障害から分離する。
	class := observability.ClassifyDatastoreError(err)
	if class == observability.DatastoreErrorClassCanceled {
		return observability.DatastoreOperationStatusCanceled, class
	}
	return observability.DatastoreOperationStatusError, class
}

func safeValkeyCommandName(cmd redis.Cmder) string {
	// Step 1: Redis command 名を lowercase へ正規化し、SigNoz facet の揺れを防ぐ。
	name := strings.TrimSpace(cmd.Name())
	if name == "" {
		return "unknown"
	}
	return strings.ToLower(name)
}

func safeValkeyCommandTarget(commandName string, args []any, surface string) string {
	// Step 1: EVAL は第 1 引数が Lua script body なので、script を読まず key count だけを target に残す。
	if commandName == "eval" || commandName == "evalsha" {
		return safeValkeyEvalTarget(commandName, args, surface)
	}

	// Step 2: 通常 command は第 2 引数を key とみなし、raw key ではなく namespace pattern へ変換する。
	if len(args) < 2 {
		return fmt.Sprintf("%s %s", strings.ToUpper(commandName), safeValkeySurfaceFallback(surface))
	}
	key, ok := args[1].(string)
	if !ok {
		return fmt.Sprintf("%s %s", strings.ToUpper(commandName), safeValkeySurfaceFallback(surface))
	}
	return fmt.Sprintf("%s %s", strings.ToUpper(commandName), SafeKeyPattern(key))
}

func safeValkeyEvalTarget(commandName string, args []any, surface string) string {
	// Step 1: EVAL/EVALSHA の numkeys を取り出し、Lua script body や argv を target に含めない。
	keyCount := 0
	if len(args) >= 3 {
		switch value := args[2].(type) {
		case int:
			keyCount = value
		case int64:
			keyCount = int(value)
		}
	}

	// Step 2: key が存在する場合も raw key は pattern 化し、script 本文は読まない。
	keyPattern := safeValkeySurfaceFallback(surface)
	if len(args) >= 4 {
		if key, ok := args[3].(string); ok {
			keyPattern = SafeKeyPattern(key)
		}
	}
	return fmt.Sprintf("%s key_count=%d key_pattern=%s", strings.ToUpper(commandName), keyCount, keyPattern)
}

// SafeKeyPattern は Valkey raw key を namespace pattern に変換する。
// session ID、token hash、challenge secret、メール由来 key などの末尾識別子は `*` に置換する。
//
// 引数:
//   - key: Valkey command に渡された raw key。
//
// 戻り値:
//   - string: SigNoz に安全に出せる key pattern。raw key そのものは返さない。
func SafeKeyPattern(key string) string {
	// Step 1: 空 key は固定 fallback にし、空属性や raw 値の代替出力を避ける。
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return valkeyUnknownTarget
	}

	// Step 2: colon namespace を segment 化し、既知の静的 segment 以外を wildcard へ置換する。
	segments := strings.Split(trimmed, ":")
	pattern := make([]string, 0, len(segments))
	for index, segment := range segments {
		pattern = append(pattern, safeKeySegment(index, segment))
	}

	// Step 3: 連続 wildcard を 1 つに畳み、token 長や key 階層の詳細を不要に漏らさない。
	return strings.Join(collapseWildcardSegments(pattern), ":")
}

func safeKeySegment(index int, segment string) string {
	// Step 1: 環境 prefix と思われる先頭 segment は固定語 prefix に置換し、任意 prefix の raw 文字列を出さない。
	normalized := strings.TrimSpace(segment)
	if normalized == "" {
		return "*"
	}
	if index == 0 && normalized != "product" && normalized != "admin" {
		return "prefix"
	}

	// Step 2: 実装が所有する namespace/function segment だけをそのまま残し、ID や hash は wildcard 化する。
	if isSafeStaticKeySegment(normalized) {
		return normalized
	}
	return "*"
}

func isSafeStaticKeySegment(segment string) bool {
	// Step 1: Product/Admin Valkey adapter が生成する固定 namespace だけを許可リスト化する。
	switch segment {
	case "product", "admin", "auth", "challenge", "reauth-session", "recovery-token", "recovery-delivery-failure", "recovery-session", "counter", "lock", "refresh", "refresh_index", "refresh_accounts", "session", "account-sessions", "operator-session", "operator-sessions", "test":
		return true
	default:
		return false
	}
}

func collapseWildcardSegments(segments []string) []string {
	// Step 1: 空 slice は unknown に寄せ、join 後の空文字を防ぐ。
	if len(segments) == 0 {
		return []string{valkeyUnknownTarget}
	}

	// Step 2: 連続 wildcard をまとめ、識別子 segment 数がそのまま漏れることを抑える。
	collapsed := make([]string, 0, len(segments))
	previousWildcard := false
	for _, segment := range segments {
		currentWildcard := segment == "*"
		if currentWildcard && previousWildcard {
			continue
		}
		collapsed = append(collapsed, segment)
		previousWildcard = currentWildcard
	}
	return collapsed
}

func safeValkeySurfaceFallback(surface string) string {
	// Step 1: surface 名は Product/Admin の固定文字列だけを出し、それ以外は unknown に寄せる。
	switch strings.TrimSpace(surface) {
	case "product", "admin":
		return strings.TrimSpace(surface)
	default:
		return valkeyUnknownTarget
	}
}

func valkeyRequestArgumentBytes(commandName string, args []any) int {
	// Step 1: EVAL/EVALSHA は Lua script body を計測対象から除外し、script 長という情報も出さない。
	startIndex := 1
	if commandName == "eval" || commandName == "evalsha" {
		startIndex = 2
	}

	// Step 2: 値そのものは記録せず、文字列表現の長さだけを合算する。
	total := 0
	for index, arg := range args {
		if index < startIndex {
			continue
		}
		if value, ok := arg.(string); ok {
			total += len(value)
		}
	}
	return total
}

func valkeyResponseBytes(cmd redis.Cmder) int {
	// Step 1: response 本文は出さず、型別に長さだけを集計する。
	switch typed := cmd.(type) {
	case *redis.StringCmd:
		value, err := typed.Result()
		if err != nil {
			return 0
		}
		return len(value)
	case *redis.StringSliceCmd:
		values, err := typed.Result()
		if err != nil {
			return 0
		}
		total := 0
		for _, value := range values {
			total += len(value)
		}
		return total
	case *redis.Cmd:
		return genericValkeyResponseBytes(typed.Val())
	default:
		return 0
	}
}

func genericValkeyResponseBytes(value any) int {
	// Step 1: EVAL などの generic result も本文ではなく size だけへ畳む。
	switch typed := value.(type) {
	case string:
		return len(typed)
	case []byte:
		return len(typed)
	case []any:
		total := 0
		for _, item := range typed {
			total += genericValkeyResponseBytes(item)
		}
		return total
	default:
		return 0
	}
}

func valkeyResultCount(cmd redis.Cmder) int {
	// Step 1: collection 型だけ件数を返し、値の内容は呼び出し元へ渡さない。
	switch typed := cmd.(type) {
	case *redis.StringSliceCmd:
		values, err := typed.Result()
		if err != nil {
			return 0
		}
		return len(values)
	case *redis.Cmd:
		if values, ok := typed.Val().([]any); ok {
			return len(values)
		}
		return 0
	default:
		return 0
	}
}

func valkeyResultClass(cmd redis.Cmder, errorClass observability.DatastoreErrorClass) observability.DatastoreResultClass {
	// Step 1: not_found は command 型に関係なく結果なしとして扱う。
	if errorClass == observability.DatastoreErrorClassNotFound {
		return observability.DatastoreResultClassNone
	}

	// Step 2: command result 型から本文なしの粗い分類だけを返す。
	switch cmd.(type) {
	case *redis.StringCmd:
		return observability.DatastoreResultClassSingle
	case *redis.StringSliceCmd:
		return observability.DatastoreResultClassCollection
	case *redis.IntCmd:
		return observability.DatastoreResultClassInteger
	case *redis.StatusCmd, *redis.BoolCmd:
		return observability.DatastoreResultClassStatus
	case *redis.Cmd:
		return observability.DatastoreResultClassUnknown
	default:
		return observability.DatastoreResultClassUnknown
	}
}

func optionalPositiveInt64(value int64) *int64 {
	// Step 1: 0 は「情報なし」として省略し、存在しない payload の 0 byte と未計測を混同しない。
	if value <= 0 {
		return nil
	}
	return &value
}
