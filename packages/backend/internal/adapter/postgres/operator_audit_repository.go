package postgres

import (
	"context"
	"crypto/rand"
	"errors"
	"time"

	"gorm.io/gorm"

	auditapplication "www-template/packages/backend/internal/application/audit"
	"www-template/packages/backend/internal/platform/id"
)

var _ auditapplication.Repository = (*OperatorAuditRepository)(nil)

// OperatorAuditRepository は Admin audit intent/outcome を admin.audit_events に保存する PostgreSQL adapter である。
//
// 役割:
//   - application 層の Repository port を実装し、GORM record や SQL detail を外へ出さない。
//   - intent 作成時に audit ID を ULID として発行し、mutation correlation を Admin schema 内へ閉じる。
//   - outcome 完了では succeeded/failed の primitive DTO だけを保存し、domain transition rule は application/domain 側へ残す。
//   - schema/table/role/grant 境界で security を守り、package path による Product/Admin 分離は行わない。
type OperatorAuditRepository struct {
	db *gorm.DB
}

type auditEventRecord struct {
	ID                 string     `gorm:"column:id;primaryKey"`
	RequestID          string     `gorm:"column:request_id"`
	OperatorID         string     `gorm:"column:operator_id"`
	TargetAccountID    string     `gorm:"column:target_account_id"`
	TargetAccountEmail string     `gorm:"column:target_account_email"`
	Action             string     `gorm:"column:action"`
	Outcome            string     `gorm:"column:outcome"`
	StableErrorCode    *string    `gorm:"column:stable_error_code"`
	Metadata           string     `gorm:"column:metadata;type:jsonb"`
	CreatedAt          time.Time  `gorm:"column:created_at"`
	CompletedAt        *time.Time `gorm:"column:completed_at"`
}

func (auditEventRecord) TableName() string {
	// Step 1: audit persistence は Admin-owned schema に固定し、Product table へ監査 event を混ぜない。
	return "admin.audit_events"
}

// NewOperatorAuditRepository は Admin audit repository を構築する。
//
// db は runtime composition で接続・ping 済みの GORM handle を渡す。
// nil handle は各 method で fail-closed に検出し、構築時には外部 I/O を行わない。
func NewOperatorAuditRepository(db *gorm.DB) *OperatorAuditRepository {
	// Step 1: DB handle を保持し、connection lifecycle は Admin runtime container が所有する。
	return &OperatorAuditRepository{db: db}
}

// RecordAuditIntent は pending audit intent を admin.audit_events に保存する。
func (r *OperatorAuditRepository) RecordAuditIntent(ctx context.Context, record auditapplication.IntentRecord) (auditapplication.Record, error) {
	// Step 1: repository または DB handle が欠ける場合は監査なし mutation を防ぐため内部 error に畳む。
	if r == nil || r.db == nil {
		return auditapplication.Record{}, auditapplication.ErrAuditInternal
	}

	// Step 2: audit ID は intent 発生時刻を使う ULID とし、後続 mutation の correlation ID として返す。
	auditID, err := id.NewULID(record.OccurredAt, rand.Reader)
	if err != nil {
		return auditapplication.Record{}, auditapplication.ErrAuditInternal
	}
	audit := auditEventRecordFromIntent(record, auditID)
	if err := r.db.WithContext(ctx).Create(&audit).Error; err != nil {
		return auditapplication.Record{}, auditapplication.ErrAuditInternal
	}
	return audit.toApplicationRecord(), nil
}

// FindAudit は audit ID に対応する Admin audit snapshot を返す。
func (r *OperatorAuditRepository) FindAudit(ctx context.Context, auditID string) (auditapplication.Record, error) {
	// Step 1: repository または DB handle が欠ける場合は復元不能として内部 error に畳む。
	if r == nil || r.db == nil {
		return auditapplication.Record{}, auditapplication.ErrAuditInternal
	}

	// Step 2: admin.audit_events だけを primary key で参照し、Product schema へ audit state を探しに行かない。
	var audit auditEventRecord
	if err := r.db.WithContext(ctx).Where("id = ?", auditID).First(&audit).Error; err != nil {
		return auditapplication.Record{}, mapAuditRepositoryError(err)
	}
	return audit.toApplicationRecord(), nil
}

// CompleteAudit は pending audit event を succeeded または failed outcome へ更新する。
func (r *OperatorAuditRepository) CompleteAudit(ctx context.Context, record auditapplication.CompletionRecord) (auditapplication.Record, error) {
	// Step 1: repository または DB handle が欠ける場合は outcome 保存不能として internal error に畳む。
	if r == nil || r.db == nil {
		return auditapplication.Record{}, auditapplication.ErrAuditInternal
	}

	// Step 2: pending 行だけを更新し、二重完了や存在しない audit ID は application の内部不整合として扱う。
	updates := auditCompletionUpdates(record)
	result := r.db.WithContext(ctx).Model(&auditEventRecord{}).Where("id = ? AND outcome = ?", record.AuditID, "pending").Updates(updates)
	if result.Error != nil || result.RowsAffected == 0 {
		return auditapplication.Record{}, auditapplication.ErrAuditInternal
	}
	return r.FindAudit(ctx, record.AuditID)
}

func auditEventRecordFromIntent(record auditapplication.IntentRecord, auditID string) auditEventRecord {
	// Step 1: metadata は空文字の場合に空 JSON object とし、JSONB の NOT NULL default と同じ意味にそろえる。
	metadata := record.DetailsJSON
	if metadata == "" {
		metadata = "{}"
	}
	return auditEventRecord{ID: auditID, RequestID: record.RequestID, OperatorID: record.OperatorID, Action: record.Action, Outcome: record.Outcome, Metadata: metadata, CreatedAt: record.OccurredAt.UTC()}
}

func auditCompletionUpdates(record auditapplication.CompletionRecord) map[string]any {
	// Step 1: stable_error_code は success では NULL、failure では stable code 文字列として保存し、DB CHECK 制約と一致させる。
	var stableErrorCode *string
	if record.StableErrorCode != "" {
		stableErrorCode = &record.StableErrorCode
	}
	return map[string]any{"outcome": record.Outcome, "stable_error_code": stableErrorCode, "completed_at": record.CompletedAt.UTC()}
}

func (r auditEventRecord) toApplicationRecord() auditapplication.Record {
	// Step 1: nullable stable_error_code を application DTO の空文字表現へ戻し、domain reconstitution が既存 rule で解釈できるようにする。
	stableErrorCode := ""
	if r.StableErrorCode != nil {
		stableErrorCode = *r.StableErrorCode
	}
	return auditapplication.Record{AuditID: r.ID, OperatorID: r.OperatorID, Action: r.Action, TargetType: "account", TargetID: r.TargetAccountID, RequestID: r.RequestID, DetailsJSON: r.Metadata, Outcome: r.Outcome, StableErrorCode: stableErrorCode, OccurredAt: r.CreatedAt, CompletedAt: r.CompletedAt}
}

func mapAuditRepositoryError(err error) error {
	// Step 1: not found も復元不能な監査状態として扱い、外部へ DB 詳細を出さない抽象 error に畳む。
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return auditapplication.ErrAuditInternal
	}
	return auditapplication.ErrAuditInternal
}
