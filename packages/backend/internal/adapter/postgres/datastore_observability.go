package postgres

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"www-template/packages/backend/internal/platform/observability"
)

const postgresStatementTargetMaxRunes = 512

var (
	postgresBlockCommentPattern = regexp.MustCompile(`(?s)/\*.*?\*/`)
	postgresLineCommentPattern  = regexp.MustCompile(`--[^\r\n]*`)
	postgresStringPattern       = regexp.MustCompile(`'(?:''|[^'])*'`)
	postgresEmailPattern        = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)
	postgresUUIDPattern         = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}\b`)
	postgresNumberPattern       = regexp.MustCompile(`\b\d+(?:\.\d+)?\b`)
)

// observedGORMLogger は GORM logger.Interface に datastore 観測を追加する wrapper である。
// delegate の Warn/Error 設定は維持し、通常 query の SigNoz 記録だけを別経路で追加する。
type observedGORMLogger struct {
	delegate logger.Interface
}

func newObservedGORMLogger(delegate logger.Interface) logger.Interface {
	// Step 1: nil delegate の場合は GORM の discard logger を使い、wrapper 呼び出しで nil panic を起こさない。
	if delegate == nil {
		delegate = logger.Discard
	}

	// Step 2: delegate のセキュリティ設定を変えず、Trace だけに datastore 観測を追加する。
	return observedGORMLogger{delegate: delegate}
}

func (l observedGORMLogger) LogMode(level logger.LogLevel) logger.Interface {
	// Step 1: GORM が log level を切り替える場合も wrapper を維持し、全 query 観測が外れないようにする。
	return observedGORMLogger{delegate: l.delegate.LogMode(level)}
}

func (l observedGORMLogger) Info(ctx context.Context, message string, values ...any) {
	// Step 1: 既存 GORM logger の Info 出力ポリシーを変更しないため、そのまま delegate へ渡す。
	l.delegate.Info(ctx, message, values...)
}

func (l observedGORMLogger) Warn(ctx context.Context, message string, values ...any) {
	// Step 1: 既存 GORM logger の Warn 出力ポリシーを変更しないため、そのまま delegate へ渡す。
	l.delegate.Warn(ctx, message, values...)
}

func (l observedGORMLogger) Error(ctx context.Context, message string, values ...any) {
	// Step 1: 既存 GORM logger の Error 出力ポリシーを変更しないため、そのまま delegate へ渡す。
	l.delegate.Error(ctx, message, values...)
}

func (l observedGORMLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	// Step 1: GORM の fc は SQL 文字列生成を含むため、一度だけ評価して delegate と観測処理で共有する。
	var statement string
	var rowsAffected int64
	resolved := false
	cachedFC := func() (string, int64) {
		if !resolved {
			statement, rowsAffected = fc()
			resolved = true
		}
		return statement, rowsAffected
	}

	// Step 2: 既存 logger の Warn/slow/error 出力を維持し、既存テストが固定する security posture を変えない。
	l.delegate.Trace(ctx, begin, cachedFC, err)

	// Step 3: delegate が fc を呼ばない log level でも観測側は全 query を記録するため、ここで必ず SQL template を取得する。
	statement, rowsAffected = cachedFC()
	observePostgresOperation(ctx, begin, statement, rowsAffected, err)
}

func observePostgresOperation(ctx context.Context, begin time.Time, statement string, rowsAffected int64, err error) {
	// Step 1: GORM 由来の SQL 文字列を再サニタイズし、ParameterizedQueries 設定に依存せず bind 値や literal を出さない。
	safeStatement := sanitizePostgresStatement(statement)
	operation := postgresOperationFromStatement(safeStatement)
	status, errorClass := postgresObservationStatus(err)

	// Step 2: rows affected は GORM が -1 を返す場合だけ未設定にし、0 行更新など意味のある 0 は保持する。
	var rowsPointer *int64
	if rowsAffected >= 0 {
		rows := rowsAffected
		rowsPointer = &rows
	}

	// Step 3: 結果本文を出さず、operation と rows から安全な結果分類だけを付与する。
	observability.ObserveDatastoreOperationCompleted(ctx, observability.DatastoreOperationCompleted{
		System:       observability.DatastoreSystemPostgreSQL,
		Operation:    operation,
		Target:       safeStatement,
		Status:       status,
		Duration:     time.Since(begin),
		RowsAffected: rowsPointer,
		ResultClass:  postgresResultClass(operation, rowsAffected, errorClass),
		ErrorClass:   errorClass,
	})
}

func observePostgresPing(ctx context.Context, begin time.Time, err error) {
	// Step 1: database/sql の PingContext は GORM logger を通らないため、startup health I/O として明示的に記録する。
	status, errorClass := postgresObservationStatus(err)
	observability.ObserveDatastoreOperationCompleted(ctx, observability.DatastoreOperationCompleted{
		System:      observability.DatastoreSystemPostgreSQL,
		Operation:   "ping",
		Target:      "postgres.ping",
		Status:      status,
		Duration:    time.Since(begin),
		ResultClass: observability.DatastoreResultClassStatus,
		ErrorClass:  errorClass,
	})
}

func postgresObservationStatus(err error) (observability.DatastoreOperationStatus, observability.DatastoreErrorClass) {
	// Step 1: 成功時は status=ok/error_class=none として、成功操作も同じ facet で検索できるようにする。
	if err == nil {
		return observability.DatastoreOperationStatusOK, observability.DatastoreErrorClassNone
	}

	// Step 2: GORM の record not found は運用上の通常結果なので、error ではなく not_found に分類する。
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return observability.DatastoreOperationStatusNotFound, observability.DatastoreErrorClassNotFound
	}

	// Step 3: context cancellation は error ではなく canceled status として切り分け、client 中断と DB 障害を混同しない。
	class := observability.ClassifyDatastoreError(err)
	if class == observability.DatastoreErrorClassCanceled {
		return observability.DatastoreOperationStatusCanceled, class
	}
	return observability.DatastoreOperationStatusError, class
}

func postgresResultClass(operation string, rowsAffected int64, errorClass observability.DatastoreErrorClass) observability.DatastoreResultClass {
	// Step 1: not_found は rows の値に関係なく結果なしとして扱う。
	if errorClass == observability.DatastoreErrorClassNotFound {
		return observability.DatastoreResultClassNone
	}

	// Step 2: SELECT 系は rows から本文なしの結果規模だけを分類する。
	if operation == "select" {
		switch {
		case rowsAffected == 0:
			return observability.DatastoreResultClassEmpty
		case rowsAffected == 1:
			return observability.DatastoreResultClassSingle
		case rowsAffected > 1:
			return observability.DatastoreResultClassCollection
		default:
			return observability.DatastoreResultClassUnknown
		}
	}

	// Step 3: mutation 系は本文ではなく DB が受理した事実だけを記録する。
	if operation == "insert" || operation == "update" || operation == "delete" {
		return observability.DatastoreResultClassAcknowledged
	}
	return observability.DatastoreResultClassUnknown
}

func postgresOperationFromStatement(statement string) string {
	// Step 1: statement 先頭 token だけを使い、SQL 全体の内容を operation facet に混ぜない。
	trimmed := strings.TrimSpace(statement)
	if trimmed == "" {
		return "unknown"
	}
	fields := strings.Fields(strings.TrimLeft(trimmed, "("))
	if len(fields) == 0 {
		return "unknown"
	}

	// Step 2: 代表的な DML を安定した lowercase operation へ正規化する。
	switch strings.ToUpper(fields[0]) {
	case "SELECT":
		return "select"
	case "INSERT":
		return "insert"
	case "UPDATE":
		return "update"
	case "DELETE":
		return "delete"
	case "WITH":
		return "select"
	default:
		return strings.ToLower(fields[0])
	}
}

func sanitizePostgresStatement(statement string) string {
	// Step 1: コメントを除去し、コメント中に紛れた PII や secret を出さない。
	sanitized := postgresBlockCommentPattern.ReplaceAllString(statement, " ")
	sanitized = postgresLineCommentPattern.ReplaceAllString(sanitized, " ")

	// Step 2: SQL literal と代表的な identifier-like secret を placeholder へ置換し、ParameterizedQueries の設定漏れにも備える。
	sanitized = postgresStringPattern.ReplaceAllString(sanitized, "?")
	sanitized = postgresEmailPattern.ReplaceAllString(sanitized, "?")
	sanitized = postgresUUIDPattern.ReplaceAllString(sanitized, "?")
	sanitized = postgresNumberPattern.ReplaceAllString(sanitized, "?")

	// Step 3: 空白を 1 つへ畳み、SigNoz facet で同じ query template がまとまるようにする。
	sanitized = strings.Join(strings.Fields(sanitized), " ")
	if sanitized == "" {
		return "unknown"
	}

	// Step 4: 非常に長い statement は安全な template でもログ量を増やすため、rune 単位で上限をかける。
	return truncatePostgresStatementTarget(sanitized)
}

func truncatePostgresStatementTarget(statement string) string {
	// Step 1: 短い statement は allocation を増やさずそのまま返す。
	if utf8.RuneCountInString(statement) <= postgresStatementTargetMaxRunes {
		return statement
	}

	// Step 2: UTF-8 を壊さないよう rune slice 化してから上限で切り、truncated marker を付ける。
	runes := []rune(statement)
	return string(runes[:postgresStatementTargetMaxRunes]) + " ...[truncated]"
}

var _ logger.Interface = observedGORMLogger{}
