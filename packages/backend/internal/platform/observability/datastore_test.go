package observability

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
)

// [OBS-DATASTORE-1] 安全 DTO から安定キーだけを構築し、raw 値用の属性を要求しないことを検証する。
func TestDatastoreOperationCompletedLogAttrsUseStableSafeKeys(t *testing.T) {
	rowsAffected := int64(2)
	requestBytes := int64(128)
	responseBytes := int64(64)

	completed := normalizeDatastoreOperationCompleted(DatastoreOperationCompleted{
		System:        DatastoreSystemPostgreSQL,
		Operation:     "query",
		Target:        "SELECT * FROM accounts WHERE email = ?",
		Status:        DatastoreOperationStatusOK,
		Duration:      250 * time.Millisecond,
		RowsAffected:  &rowsAffected,
		RequestBytes:  &requestBytes,
		ResponseBytes: &responseBytes,
		ResultClass:   DatastoreResultClassSingle,
	})

	attrs := attrsToMap(completed.logAttrs())

	if got := attrs["event_type"].String(); got != DatastoreOperationCompletedEventType {
		t.Fatalf("event_type = %q, want %q", got, DatastoreOperationCompletedEventType)
	}
	if got := attrs["datastore.system"].String(); got != string(DatastoreSystemPostgreSQL) {
		t.Fatalf("datastore.system = %q, want %q", got, DatastoreSystemPostgreSQL)
	}
	if got := attrs["datastore.operation"].String(); got != "query" {
		t.Fatalf("datastore.operation = %q, want query", got)
	}
	if got := attrs["datastore.target"].String(); got != "SELECT * FROM accounts WHERE email = ?" {
		t.Fatalf("datastore.target = %q, want safe SQL template", got)
	}
	if got := attrs["datastore.status"].String(); got != string(DatastoreOperationStatusOK) {
		t.Fatalf("datastore.status = %q, want %q", got, DatastoreOperationStatusOK)
	}
	if got := attrs["duration_ms"].Int64(); got != 250 {
		t.Fatalf("duration_ms = %d, want 250", got)
	}
	if got := attrs["rows_affected"].Int64(); got != rowsAffected {
		t.Fatalf("rows_affected = %d, want %d", got, rowsAffected)
	}
	if got := attrs["request_bytes"].Int64(); got != requestBytes {
		t.Fatalf("request_bytes = %d, want %d", got, requestBytes)
	}
	if got := attrs["response_bytes"].Int64(); got != responseBytes {
		t.Fatalf("response_bytes = %d, want %d", got, responseBytes)
	}
	if got := attrs["result_class"].String(); got != string(DatastoreResultClassSingle) {
		t.Fatalf("result_class = %q, want %q", got, DatastoreResultClassSingle)
	}
	if got := attrs["error_class"].String(); got != string(DatastoreErrorClassNone) {
		// Step 1: 成功操作も error_class=none を持ち、SigNoz facet で成功/失敗を同じキーで検索できることを確認する。
		t.Fatalf("error_class = %q, want %q", got, DatastoreErrorClassNone)
	}
	if got := attrs["raw_value_logged"].Bool(); got {
		t.Fatal("raw_value_logged = true, want false")
	}
	if _, exists := attrs["sql.bind_values"]; exists {
		t.Fatal("unexpected raw SQL bind attribute present")
	}
	if _, exists := attrs["datastore.raw_key"]; exists {
		t.Fatal("unexpected raw key attribute present")
	}
}

// [OBS-DATASTORE-2] 観測 helper が slog 出力と trace 互換属性を同じ安全キーで構築することを検証する。
func TestObserveDatastoreOperationCompletedEmitsStructuredLogAndTraceCompatibleAttrs(t *testing.T) {
	previousLogger := slog.Default()
	logHandler := &captureHandler{}
	slog.SetDefault(slog.New(logHandler))
	defer slog.SetDefault(previousLogger)

	requestBytes := int64(17)
	ObserveDatastoreOperationCompleted(context.Background(), DatastoreOperationCompleted{
		System:       DatastoreSystemValkey,
		Operation:    "get",
		Target:       "GET account_session",
		Status:       DatastoreOperationStatusError,
		Duration:     12 * time.Millisecond,
		RequestBytes: &requestBytes,
		ErrorClass:   DatastoreErrorClassNotFound,
		ResultClass:  DatastoreResultClassNone,
	})

	if len(logHandler.records) != 1 {
		t.Fatalf("log record count = %d, want 1", len(logHandler.records))
	}
	logRecord := logHandler.records[0]
	if logRecord.message != "datastore operation completed" {
		t.Fatalf("log message = %q, want datastore operation completed", logRecord.message)
	}
	logAttrs := attrsToMap(logRecord.attrs)
	if got := logAttrs["event_type"].String(); got != DatastoreOperationCompletedEventType {
		t.Fatalf("log event_type = %q, want %q", got, DatastoreOperationCompletedEventType)
	}
	if got := logAttrs["error_class"].String(); got != string(DatastoreErrorClassNotFound) {
		t.Fatalf("log error_class = %q, want %q", got, DatastoreErrorClassNotFound)
	}
	if got := logAttrs["raw_value_logged"].Bool(); got {
		t.Fatal("log raw_value_logged = true, want false")
	}

	eventAttrs := traceAttrsToMap(slogAttrsToTraceAttributes(logRecord.attrs))
	if got := eventAttrs["datastore.system"].AsString(); got != string(DatastoreSystemValkey) {
		t.Fatalf("trace datastore.system = %q, want %q", got, DatastoreSystemValkey)
	}
	if got := eventAttrs["error_class"].AsString(); got != string(DatastoreErrorClassNotFound) {
		t.Fatalf("trace error_class = %q, want %q", got, DatastoreErrorClassNotFound)
	}
	if got := eventAttrs["raw_value_logged"].AsBool(); got {
		t.Fatal("trace raw_value_logged = true, want false")
	}
}

// [OBS-DATASTORE-3] error 分類 helper が raw error 文言に依存せず安全な class を返すことを検証する。
func TestClassifyDatastoreError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected DatastoreErrorClass
	}{
		{name: "nil returns none", err: nil, expected: DatastoreErrorClassNone},
		{name: "context canceled", err: context.Canceled, expected: DatastoreErrorClassCanceled},
		{name: "deadline exceeded", err: context.DeadlineExceeded, expected: DatastoreErrorClassDeadlineExceeded},
		{name: "wrapped not found", err: WrapDatastoreNotFoundError(errors.New("not found")), expected: DatastoreErrorClassNotFound},
		{name: "wrapped unexpected status", err: WrapDatastoreUnexpectedStatusError(errors.New("status mismatch")), expected: DatastoreErrorClassUnexpectedStatus},
		{name: "network error", err: &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("boom")}, expected: DatastoreErrorClassConnectionError},
		{name: "unknown error", err: errors.New("boom"), expected: DatastoreErrorClassUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyDatastoreError(tt.err)
			if got != tt.expected {
				t.Fatalf("ClassifyDatastoreError() = %q, want %q", got, tt.expected)
			}
		})
	}
}

type capturedRecord struct {
	message string
	attrs   []slog.Attr
}

type captureHandler struct {
	records []capturedRecord
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *captureHandler) Handle(_ context.Context, record slog.Record) error {
	attrs := make([]slog.Attr, 0, record.NumAttrs())
	record.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)
		return true
	})
	h.records = append(h.records, capturedRecord{message: record.Message, attrs: attrs})
	return nil
}

func (h *captureHandler) WithAttrs([]slog.Attr) slog.Handler {
	return h
}

func (h *captureHandler) WithGroup(string) slog.Handler {
	return h
}

func attrsToMap(attrs []slog.Attr) map[string]slog.Value {
	attrMap := make(map[string]slog.Value, len(attrs))
	for _, attr := range attrs {
		attrMap[attr.Key] = attr.Value.Resolve()
	}
	return attrMap
}

func traceAttrsToMap(attrs []attribute.KeyValue) map[string]attribute.Value {
	attrMap := make(map[string]attribute.Value, len(attrs))
	for _, attr := range attrs {
		attrMap[string(attr.Key)] = attr.Value
	}
	return attrMap
}
