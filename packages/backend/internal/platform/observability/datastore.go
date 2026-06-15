package observability

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// DatastoreOperationCompletedEventType は datastore 操作完了イベントの安定 event_type である。
// PostgreSQL、Valkey、OpenSearch の各 adapter は、この event_type を前提に SigNoz 上で横断検索できる。
const DatastoreOperationCompletedEventType = "datastore.operation.completed"

// DatastoreSystem は観測対象 datastore の種類を表す。
// adapter 層は OTel 固有型を知らずに、この文字列型だけで system 名を指定する。
type DatastoreSystem string

const (
	// DatastoreSystemUnknown は system が未指定だった場合の安全なフォールバック値である。
	DatastoreSystemUnknown DatastoreSystem = "unknown"
	// DatastoreSystemPostgreSQL は PostgreSQL 操作を表す。
	DatastoreSystemPostgreSQL DatastoreSystem = "postgresql"
	// DatastoreSystemValkey は Valkey 操作を表す。
	DatastoreSystemValkey DatastoreSystem = "valkey"
	// DatastoreSystemOpenSearch は OpenSearch 操作を表す。
	DatastoreSystemOpenSearch DatastoreSystem = "opensearch"
)

// DatastoreOperationStatus は datastore 操作結果の大分類である。
// 詳細な失敗理由は error_class に寄せ、status は検索しやすい少数の安定値に保つ。
type DatastoreOperationStatus string

const (
	// DatastoreOperationStatusOK は操作成功を表す。
	DatastoreOperationStatusOK DatastoreOperationStatus = "ok"
	// DatastoreOperationStatusNotFound は datastore が正常応答したが対象が存在しなかったことを表す。
	DatastoreOperationStatusNotFound DatastoreOperationStatus = "not_found"
	// DatastoreOperationStatusCanceled は caller cancellation により操作が完了しなかったことを表す。
	DatastoreOperationStatusCanceled DatastoreOperationStatus = "canceled"
	// DatastoreOperationStatusError は操作失敗を表す。
	DatastoreOperationStatusError DatastoreOperationStatus = "error"
)

// DatastoreResultClass は payload 本文を出さずに結果の形だけを記録する分類値である。
// caller は件数や本文ではなく、「0 件」「単一結果」などの粗い分類だけを渡す。
type DatastoreResultClass string

const (
	// DatastoreResultClassEmpty は結果 payload が空だったことを表す。
	DatastoreResultClassEmpty DatastoreResultClass = "empty"
	// DatastoreResultClassNone は該当結果が存在しないことを表す。
	DatastoreResultClassNone DatastoreResultClass = "none"
	// DatastoreResultClassSingle は単一結果を表す。
	DatastoreResultClassSingle DatastoreResultClass = "single"
	// DatastoreResultClassCollection は複数要素を持つ collection 結果を表す。
	DatastoreResultClassCollection DatastoreResultClass = "collection"
	// DatastoreResultClassMultiple は複数結果を表す。
	DatastoreResultClassMultiple DatastoreResultClass = "multiple"
	// DatastoreResultClassInteger は Valkey の INCR/DEL など整数結果を表す。
	DatastoreResultClassInteger DatastoreResultClass = "integer"
	// DatastoreResultClassStatus は Valkey の OK や OpenSearch の status のような状態応答を表す。
	DatastoreResultClassStatus DatastoreResultClass = "status"
	// DatastoreResultClassAcknowledged は Valkey/OpenSearch などの副作用完了だけが重要な操作を表す。
	DatastoreResultClassAcknowledged DatastoreResultClass = "acknowledged"
	// DatastoreResultClassUnknown は安全な結果分類を caller が決定できなかったことを表す。
	DatastoreResultClassUnknown DatastoreResultClass = "unknown"
)

// DatastoreErrorClass は datastore 失敗を安定した粗い分類へ正規化した値である。
// raw error text ではなく、この分類だけを trace/log 属性へ出すことで秘匿情報漏えいを防ぐ。
type DatastoreErrorClass string

const (
	// DatastoreErrorClassNone は成功またはエラーなしを表す。
	DatastoreErrorClassNone DatastoreErrorClass = "none"
	// DatastoreErrorClassCanceled は caller 側 cancellation による中断を表す。
	DatastoreErrorClassCanceled DatastoreErrorClass = "canceled"
	// DatastoreErrorClassDeadlineExceeded は timeout 超過を表す。
	DatastoreErrorClassDeadlineExceeded DatastoreErrorClass = "deadline_exceeded"
	// DatastoreErrorClassNotFound は結果不在を表す。
	DatastoreErrorClassNotFound DatastoreErrorClass = "not_found"
	// DatastoreErrorClassUnexpectedStatus は OpenSearch/HTTP などの予期しない status を表す。
	DatastoreErrorClassUnexpectedStatus DatastoreErrorClass = "unexpected_status"
	// DatastoreErrorClassConnectionError は接続失敗やソケット異常を表す。
	DatastoreErrorClassConnectionError DatastoreErrorClass = "connection_error"
	// DatastoreErrorClassUnknown は上記のどれにも該当しない失敗を表す。
	DatastoreErrorClassUnknown DatastoreErrorClass = "unknown"
)

// DatastoreOperationCompleted は安全な datastore 操作完了メタデータだけを保持する DTO である。
// SQL bind 値、Valkey value、raw key、Lua script body、OpenSearch body、PII、secret はこの DTO に含めない。
// Target には raw identifier ではなく SQL template、command 名、index template などの安全な識別子だけを入れる。
type DatastoreOperationCompleted struct {
	// System は PostgreSQL / Valkey / OpenSearch のどの系統かを示す。
	System DatastoreSystem
	// Operation は query / exec / get / set / request などの安定した操作名を示す。
	Operation string
	// Target は SQL template、Valkey command 名、OpenSearch endpoint template などの安全な対象名を示す。
	Target string
	// Status は成功か失敗かの大分類である。
	Status DatastoreOperationStatus
	// Duration は I/O 完了までの経過時間である。
	Duration time.Duration
	// RowsAffected は更新系 SQL の影響行数など、存在する場合だけ設定する。
	RowsAffected *int64
	// ResultCount は検索ヒット数や配列要素数など、本文を出さず件数だけを記録したい場合に設定する。
	ResultCount *int64
	// RequestBytes は body 長や送信 payload 長など、安全に計測できるサイズだけを記録する。
	RequestBytes *int64
	// ResponseBytes は response body 長や受信 payload 長など、安全に計測できるサイズだけを記録する。
	ResponseBytes *int64
	// StatusCode は OpenSearch など HTTP 系 datastore の response status code を記録する。
	StatusCode *int64
	// ResultClass は none / single / multiple など、本文を含まない結果分類を示す。
	ResultClass DatastoreResultClass
	// ErrorClass は失敗時の安定分類だけを示す。raw error message は出さない。
	ErrorClass DatastoreErrorClass
}

// ObserveDatastoreOperationCompleted は安全な datastore 操作完了イベントを slog と trace event の両方へ記録する。
// adapter 層はこの API を呼ぶだけでよく、OTel 型や attribute 構築を直接扱う必要がない。
// 入力 DTO に raw 値を持たせない設計にすることで、機微情報の混入余地を API 形状自体で減らす。
//
// 引数:
//   - ctx: trace 相関と logger 相関を引き継ぐ context。
//   - completed: 安全なメタデータだけを持つ datastore 操作完了 DTO。
//
// 副作用:
//   - observability.Logger() 経由で構造化ログを 1 件出力する。
//   - 現在の span が recording 中であれば trace event を 1 件追加する。
func ObserveDatastoreOperationCompleted(ctx context.Context, completed DatastoreOperationCompleted) {
	// Step 1: caller から渡された DTO を正規化し、空欄や負値があっても安定キーで記録できる形にする。
	normalized := normalizeDatastoreOperationCompleted(completed)

	// Step 2: slog/trace の両方で使い回す属性配列を一度だけ構築し、検索キーのズレを防ぐ。
	attrs := normalized.logAttrs()

	// Step 3: 失敗時だけ warning に上げ、それ以外は info で安定した運用ノイズに保つ。
	Logger().LogAttrs(ctx, normalized.logLevel(), "datastore operation completed", attrs...)

	// Step 4: 同じ属性セットを親 span の event にも付与し、HTTP request span から datastore 操作を横断検索できるようにする。
	AddTraceEvent(ctx, DatastoreOperationCompletedEventType, attrs)

	// Step 5: 完了済み操作を duration 付きの child span としても残し、SigNoz の trace waterfall で I/O 時間を確認できるようにする。
	ObserveCompletedDatastoreSpan(ctx, normalized.spanName(), normalized.Duration, attrs)
}

// ObserveCompletedDatastoreSpan は完了済み datastore 操作を duration 付き child span として記録する。
// adapter 層は OTel 型を import せず、slog.Attr だけを渡して trace 属性へ変換できる。
//
// 引数:
//   - ctx: parent span を含む可能性がある context。
//   - spanName: SigNoz に表示する datastore span 名。空の場合は datastore.operation を使う。
//   - duration: 実際の I/O 経過時間。負値は 0 に丸める。
//   - attrs: span attribute として付与する安全な属性。raw 値を含めてはならない。
//
// 副作用:
//   - OTel global tracer provider が初期化済みの場合、child span を 1 件作成して終了する。
func ObserveCompletedDatastoreSpan(ctx context.Context, spanName string, duration time.Duration, attrs []slog.Attr) {
	// Step 1: span 名の空欄を固定値へ寄せ、SigNoz 上で無名 span が増えないようにする。
	name := strings.TrimSpace(spanName)
	if name == "" {
		name = "datastore.operation"
	}

	// Step 2: duration の負値は計測不備として 0 に丸め、終了時刻が開始時刻より前にならないようにする。
	spanDuration := duration
	if spanDuration < 0 {
		spanDuration = 0
	}

	// Step 3: 完了後に呼ばれる API なので、現在時刻から duration を引いた時刻を span 開始時刻として使う。
	endTime := time.Now()
	startTime := endTime.Add(-spanDuration)

	// Step 4: slog 属性を OTel 属性へ変換し、log と trace span で同じ key を検索できるようにする。
	_, span := otel.Tracer("www-template-datastore").Start(ctx, name,
		trace.WithTimestamp(startTime),
		trace.WithAttributes(slogAttrsToTraceAttributes(attrs)...),
	)

	// Step 5: span 内にも完了 event を付け、span 一覧と event 検索の両方で同じ情報を見られるようにする。
	span.AddEvent(DatastoreOperationCompletedEventType, trace.WithAttributes(slogAttrsToTraceAttributes(attrs)...))

	// Step 6: 終了時刻を実 I/O の完了時刻へ合わせ、trace waterfall の duration を実測値に近づける。
	span.End(trace.WithTimestamp(endTime))
}

// ClassifyDatastoreError は datastore error を安定した error_class へ正規化する。
// raw error 文言には依存せず、context 系 error・明示マーカー・network error の順で判定する。
// nil を渡した場合は none を返し、成功パスでも error_class facet を安定化する。
//
// 引数:
//   - err: datastore 操作で得た error。nil 可。
//
// 戻り値:
//   - DatastoreErrorClass: 観測属性へ安全に出せる安定分類。nil の場合は none。
func ClassifyDatastoreError(err error) DatastoreErrorClass {
	// Step 1: 成功パスでは error_class を none にし、SigNoz facet で成功操作を検索しやすくする。
	if err == nil {
		return DatastoreErrorClassNone
	}

	// Step 2: cancellation と timeout は最優先で判定し、network error へ誤分類しないようにする。
	if errors.Is(err, context.Canceled) {
		return DatastoreErrorClassCanceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return DatastoreErrorClassDeadlineExceeded
	}

	// Step 3: caller が明示的に付けた分類があれば、それを最優先で採用する。
	var classifier datastoreErrorClassifier
	if errors.As(err, &classifier) {
		classified := normalizeDatastoreErrorClass(classifier.DatastoreErrorClass())
		if classified != "" {
			return classified
		}
	}

	// Step 4: socket / DNS / transport error は net.Error 系として connection_error に寄せる。
	var networkError net.Error
	if errors.As(err, &networkError) {
		return DatastoreErrorClassConnectionError
	}

	// Step 5: 上記以外は raw 文言に触れず unknown へ寄せる。
	return DatastoreErrorClassUnknown
}

// WrapDatastoreErrorClass は任意の error に安定した datastore error_class を付与する。
// PostgreSQL/Valkey/OpenSearch adapter は本文を晒さず、失敗理由の粗い分類だけを後段観測へ渡せる。
//
// 引数:
//   - err: 元の error。nil の場合は nil を返す。
//   - class: 付与する安定分類。
//
// 戻り値:
//   - error: errors.Is/errors.As で元 error と分類の両方を追跡できる wrapper。
func WrapDatastoreErrorClass(err error, class DatastoreErrorClass) error {
	// Step 1: nil error はそのまま返し、成功パスへ不要な wrapper を作らない。
	if err == nil {
		return nil
	}

	// Step 2: none 分類は wrapper の意味がないため、そのまま元 error を返す。
	normalized := normalizeDatastoreErrorClass(class)
	if normalized == "" || normalized == DatastoreErrorClassNone {
		return err
	}

	// Step 3: 安定分類を保持する軽量 wrapper を返し、呼び出し元が raw error text を解析しなくて済むようにする。
	return datastoreClassifiedError{class: normalized, err: err}
}

// WrapDatastoreNotFoundError は error を not_found として分類する。
// PostgreSQL の 0 件結果や Valkey の key 不在など、本文ではなく「見つからなかった」という事実だけを観測へ渡したい場面で使う。
//
// 引数:
//   - err: not found として扱いたい元 error。nil の場合は nil を返す。
//
// 戻り値:
//   - error: not_found 分類を保持した wrapper。
func WrapDatastoreNotFoundError(err error) error {
	return WrapDatastoreErrorClass(err, DatastoreErrorClassNotFound)
}

// WrapDatastoreUnexpectedStatusError は error を unexpected_status として分類する。
// OpenSearch や HTTP 系 datastore で期待レンジ外の status code を受けたときに、raw response body を出さず分類だけを残せる。
//
// 引数:
//   - err: unexpected status として扱いたい元 error。nil の場合は nil を返す。
//
// 戻り値:
//   - error: unexpected_status 分類を保持した wrapper。
func WrapDatastoreUnexpectedStatusError(err error) error {
	return WrapDatastoreErrorClass(err, DatastoreErrorClassUnexpectedStatus)
}

// WrapDatastoreConnectionError は error を connection_error として分類する。
// dial failure、socket close、transport 異常など、接続経路の問題を呼び出し側が明示したい場合に使う。
//
// 引数:
//   - err: connection error として扱いたい元 error。nil の場合は nil を返す。
//
// 戻り値:
//   - error: connection_error 分類を保持した wrapper。
func WrapDatastoreConnectionError(err error) error {
	return WrapDatastoreErrorClass(err, DatastoreErrorClassConnectionError)
}

type normalizedDatastoreOperationCompleted struct {
	System        DatastoreSystem
	Operation     string
	Target        string
	Status        DatastoreOperationStatus
	Duration      time.Duration
	RowsAffected  *int64
	ResultCount   *int64
	RequestBytes  *int64
	ResponseBytes *int64
	StatusCode    *int64
	ResultClass   DatastoreResultClass
	ErrorClass    DatastoreErrorClass
}

type datastoreErrorClassifier interface {
	DatastoreErrorClass() DatastoreErrorClass
}

type datastoreClassifiedError struct {
	class DatastoreErrorClass
	err   error
}

func (e datastoreClassifiedError) Error() string {
	return e.err.Error()
}

func (e datastoreClassifiedError) Unwrap() error {
	return e.err
}

func (e datastoreClassifiedError) DatastoreErrorClass() DatastoreErrorClass {
	return e.class
}

func normalizeDatastoreOperationCompleted(completed DatastoreOperationCompleted) normalizedDatastoreOperationCompleted {
	// Step 1: system は空欄を unknown に寄せ、SigNoz の facet を安定化する。
	system := completed.System
	if strings.TrimSpace(string(system)) == "" {
		system = DatastoreSystemUnknown
	}

	// Step 2: operation と target は空欄を安全な固定値へ正規化し、空文字 facet を避ける。
	operation := strings.TrimSpace(completed.Operation)
	if operation == "" {
		operation = "unknown"
	}
	target := strings.TrimSpace(completed.Target)
	if target == "" {
		target = "unspecified"
	}

	// Step 3: status は error_class も考慮して補完し、呼び出し側の記述漏れに耐える。
	errorClass := normalizeDatastoreErrorClass(completed.ErrorClass)
	status := normalizeDatastoreOperationStatus(completed.Status, errorClass)
	errorClass = normalizeDatastoreStatusErrorClass(status, errorClass)

	// Step 4: duration の負値は計測不備として 0 に丸め、負の milliseconds を残さない。
	duration := completed.Duration
	if duration < 0 {
		duration = 0
	}

	// Step 5: 結果分類は空白を削り、bytes/count 系はそのまま保持して存在時だけ出力する。
	return normalizedDatastoreOperationCompleted{
		System:        system,
		Operation:     operation,
		Target:        target,
		Status:        status,
		Duration:      duration,
		RowsAffected:  completed.RowsAffected,
		ResultCount:   completed.ResultCount,
		RequestBytes:  completed.RequestBytes,
		ResponseBytes: completed.ResponseBytes,
		StatusCode:    completed.StatusCode,
		ResultClass:   DatastoreResultClass(strings.TrimSpace(string(completed.ResultClass))),
		ErrorClass:    errorClass,
	}
}

func (completed normalizedDatastoreOperationCompleted) logLevel() slog.Level {
	// Step 1: error/canceled status だけ warning に上げ、成功・not_found 系観測を Error ノイズで汚さない。
	if completed.Status == DatastoreOperationStatusError || completed.Status == DatastoreOperationStatusCanceled {
		return slog.LevelWarn
	}
	return slog.LevelInfo
}

func (completed normalizedDatastoreOperationCompleted) spanName() string {
	// Step 1: datastore system ごとに SigNoz waterfall 上の span 名を固定し、operation 名の揺れで facet が散らばらないようにする。
	switch completed.System {
	case DatastoreSystemPostgreSQL:
		return "db.query"
	case DatastoreSystemValkey:
		return "valkey.command"
	case DatastoreSystemOpenSearch:
		return "opensearch.request"
	default:
		return "datastore.operation"
	}
}

func (completed normalizedDatastoreOperationCompleted) logAttrs() []slog.Attr {
	// Step 1: 安定キーを先頭に固定し、後段検索が system/order に依存しないようにする。
	attrs := []slog.Attr{
		slog.String("event_type", DatastoreOperationCompletedEventType),
		slog.String("datastore.system", string(completed.System)),
		slog.String("datastore.operation", completed.Operation),
		slog.String("datastore.target", completed.Target),
		slog.String("datastore.status", string(completed.Status)),
		slog.Int64("duration_ms", completed.Duration.Milliseconds()),
		slog.Bool("raw_value_logged", false),
	}

	// Step 2: 任意メタデータは存在時だけ追加し、0 値と未設定値を区別できる形に保つ。
	appendOptionalInt64Attr(&attrs, "rows_affected", completed.RowsAffected)
	appendOptionalInt64Attr(&attrs, "result_count", completed.ResultCount)
	appendOptionalInt64Attr(&attrs, "request_bytes", completed.RequestBytes)
	appendOptionalInt64Attr(&attrs, "response_bytes", completed.ResponseBytes)
	appendOptionalInt64Attr(&attrs, "status_code", completed.StatusCode)
	appendOptionalStringAttr(&attrs, "result_class", string(completed.ResultClass))
	appendOptionalStringAttr(&attrs, "error_class", string(completed.ErrorClass))

	return attrs
}

func normalizeDatastoreErrorClass(class DatastoreErrorClass) DatastoreErrorClass {
	// Step 1: 空白だけの分類は未設定として扱い、不要な空文字属性を避ける。
	normalized := strings.TrimSpace(string(class))
	if normalized == "" {
		return DatastoreErrorClassNone
	}
	return DatastoreErrorClass(normalized)
}

func normalizeDatastoreOperationStatus(status DatastoreOperationStatus, errorClass DatastoreErrorClass) DatastoreOperationStatus {
	// Step 1: caller が明示した status は尊重し、adapter ごとの業務的な成功/不在判定を上書きしない。
	if strings.TrimSpace(string(status)) != "" {
		return status
	}

	// Step 2: status 未指定時は error_class から検索しやすい大分類を補完する。
	switch errorClass {
	case DatastoreErrorClassNone:
		return DatastoreOperationStatusOK
	case DatastoreErrorClassNotFound:
		return DatastoreOperationStatusNotFound
	case DatastoreErrorClassCanceled:
		return DatastoreOperationStatusCanceled
	default:
		return DatastoreOperationStatusError
	}
}

func normalizeDatastoreStatusErrorClass(status DatastoreOperationStatus, errorClass DatastoreErrorClass) DatastoreErrorClass {
	// Step 1: status が失敗系なのに error_class が none の場合は unknown へ寄せ、失敗ログの分類漏れを避ける。
	if status == DatastoreOperationStatusError && errorClass == DatastoreErrorClassNone {
		return DatastoreErrorClassUnknown
	}

	// Step 2: not_found/canceled status は対応する error_class を補い、status と class の facet を揃える。
	if status == DatastoreOperationStatusNotFound && errorClass == DatastoreErrorClassNone {
		return DatastoreErrorClassNotFound
	}
	if status == DatastoreOperationStatusCanceled && errorClass == DatastoreErrorClassNone {
		return DatastoreErrorClassCanceled
	}
	return errorClass
}

func appendOptionalInt64Attr(attrs *[]slog.Attr, key string, value *int64) {
	// Step 1: nil は未設定として扱い、0 が意味を持つケースを保つ。
	if value == nil {
		return
	}

	// Step 2: 安定キーに数値属性を追加する。
	*attrs = append(*attrs, slog.Int64(key, *value))
}

func appendOptionalStringAttr(attrs *[]slog.Attr, key string, value string) {
	// Step 1: 空白だけの文字列は未設定として扱い、ノイズ属性を増やさない。
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return
	}

	// Step 2: 安定キーに正規化済み文字列属性を追加する。
	*attrs = append(*attrs, slog.String(key, normalized))
}
