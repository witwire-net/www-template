package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	auditapplication "www-template/packages/backend/internal/application/audit"
	"www-template/packages/backend/internal/platform/config"
	"www-template/packages/backend/internal/platform/observability"
)

const operatorAuditProjectionTimeout = 3 * time.Second

type operatorAuditOpenSearchProjector struct {
	baseURL string
	prefix  string
	client  *http.Client
}

type operatorAuditProjectionWarningObserver struct {
	logger *slog.Logger
}

// NewOperatorAuditOpenSearchProjector は Operator audit event を Admin OpenSearch namespace へ投影する projector を生成する。
//
// 引数:
//   - cfg: OpenSearch 接続 URL、Admin audit prefix、Product prefix を含む設定。prefix 衝突は config validation と constructor の両方で拒否する。
//
// 戻り値:
//   - auditapplication.Projector: application use case へ注入できる projection port 実装。
//   - error: URL 欠落、prefix 欠落、Admin/Product prefix 衝突など、startup で停止すべき設定不備。
func NewOperatorAuditOpenSearchProjector(cfg config.OpenSearchConfig) (auditapplication.Projector, error) {
	// Step 1: startup validation と同じ namespace 分離を projector 単体でも確認し、単体生成時の fail-open を避ける。
	baseURL, adminPrefix, err := validateOperatorAuditOpenSearchProjectionConfig(cfg)
	if err != nil {
		return nil, err
	}

	// Step 2: timeout 付き HTTP client を保持し、OpenSearch 障害時に request goroutine が無期限に残らないようにする。
	return &operatorAuditOpenSearchProjector{baseURL: baseURL, prefix: adminPrefix, client: &http.Client{Timeout: operatorAuditProjectionTimeout}}, nil
}

// NewOperatorAuditProjectionWarningObserver は Operator audit projection failure を warning log として観測する observer を生成する。
//
// 戻り値:
//   - auditapplication.ProjectionFailureObserver: application use case に注入する failure observer。
//
// 利用例:
//
//	observer := app.NewOperatorAuditProjectionWarningObserver()
func NewOperatorAuditProjectionWarningObserver() auditapplication.ProjectionFailureObserver {
	// Step 1: 既存 observability logger を使い、projection failure を metric/log pipeline へ収集できる warning として出す。
	return &operatorAuditProjectionWarningObserver{logger: observability.Logger()}
}

func (p *operatorAuditOpenSearchProjector) ProjectOperatorAuditEvent(ctx context.Context, record auditapplication.ProjectionRecord) error {
	// Step 1: 呼び出し元 request の cancellation を尊重し、既に中断済みなら OpenSearch I/O を開始しない。
	if err := ctx.Err(); err != nil {
		return err
	}

	// Step 2: Admin audit prefix から月次 index 名を作り、Product prefix を受け付ける外部入力経路を持たないようにする。
	indexName, err := operatorAuditProjectionIndexName(p.prefix, record.OccurredAt)
	if err != nil {
		return err
	}

	// Step 3: application DTO を OpenSearch document JSON に変換し、secret や handler error text を新規に混ぜない。
	body, err := operatorAuditProjectionBody(record)
	if err != nil {
		return err
	}

	// Step 4: document ID は audit ID に固定し、retry 時も同じ document を upsert する idempotent な PUT にする。
	requestURL := p.baseURL + "/" + url.PathEscape(indexName) + "/_doc/" + url.PathEscape(record.AuditID)
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, requestURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build operator audit opensearch request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	// Step 5: 2xx 以外は projection failure として caller に返し、application 側 observer が warning/retry marker に変換できるようにする。
	response, err := p.client.Do(request)
	if err != nil {
		return fmt.Errorf("index operator audit opensearch document: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()
	_, _ = io.Copy(io.Discard, response.Body)
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("index operator audit opensearch document: unexpected status %d", response.StatusCode)
	}
	return nil
}

func (o *operatorAuditProjectionWarningObserver) ObserveOperatorAuditProjectionFailure(ctx context.Context, auditID string, err error) {
	// Step 1: projection failure は mutation rollback ではなく運用観測対象なので、warning level と stable event_type で記録する。
	logger := o.logger
	if logger == nil {
		logger = slog.Default()
	}
	logger.WarnContext(ctx, "operator audit OpenSearch projection failed", slog.String("event_type", "operator_audit.opensearch_projection_failed"), slog.String("audit_id", auditID), slog.String("error", err.Error()))
}

func validateOperatorAuditOpenSearchProjectionConfig(cfg config.OpenSearchConfig) (string, string, error) {
	// Step 1: OpenSearch URL を正規化し、空文字や path だけの値を projection 先として使わない。
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.URL), "/")
	if baseURL == "" {
		return "", "", errors.New("opensearch.url is required")
	}

	// Step 2: Admin / Product prefix を同時に必須化し、namespace 分離を知らない projector 構成を拒否する。
	adminPrefix := strings.TrimSpace(cfg.OperatorAuditIndexPrefix)
	productPrefix := strings.TrimSpace(cfg.ProductIndexPrefix)
	if adminPrefix == "" || productPrefix == "" {
		return "", "", errors.New("opensearch operator audit and product index prefixes are required")
	}

	// Step 3: startup validation と同じ同一・包含関係を拒否し、Admin projector が Product namespace へ誤投影しないことを保証する。
	adminComparable := strings.ToLower(adminPrefix)
	productComparable := strings.ToLower(productPrefix)
	if adminComparable == productComparable || strings.Contains(adminComparable, productComparable) || strings.Contains(productComparable, adminComparable) {
		return "", "", errors.New("opensearch operator audit prefix collides with product prefix")
	}
	return baseURL, adminPrefix, nil
}

func operatorAuditProjectionIndexName(prefix string, occurredAt time.Time) (string, error) {
	// Step 1: audit 発生時刻がない document は月次 index を決定できないため、OpenSearch へ送らず failure observer へ渡せる error にする。
	if occurredAt.IsZero() {
		return "", errors.New("operator audit occurred_at is required for projection")
	}

	// Step 2: UTC の年月で index を固定し、timezone による月跨ぎの namespace ずれを防ぐ。
	utc := occurredAt.UTC()
	return fmt.Sprintf("%s-%04d.%02d", prefix, utc.Year(), int(utc.Month())), nil
}

func operatorAuditProjectionBody(record auditapplication.ProjectionRecord) ([]byte, error) {
	// Step 1: DetailsJSON は保存済み JSON の場合だけ object として入れ、空の場合は null 相当の省略値にする。
	document := map[string]any{
		"id":                record.AuditID,
		"operator_id":       record.OperatorID,
		"action":            record.Action,
		"target_type":       record.TargetType,
		"target_id":         record.TargetID,
		"request_id":        record.RequestID,
		"outcome":           record.Outcome,
		"stable_error_code": record.StableErrorCode,
		"created_at":        record.OccurredAt.UTC().Format(time.RFC3339Nano),
	}
	if record.CompletedAt != nil {
		document["completed_at"] = record.CompletedAt.UTC().Format(time.RFC3339Nano)
	}
	if strings.TrimSpace(record.DetailsJSON) != "" {
		document["details_json"] = record.DetailsJSON
	}

	// Step 2: JSON encode の失敗は projection failure として扱い、mutation 自体の成功可否から切り離す。
	return json.Marshal(document)
}
