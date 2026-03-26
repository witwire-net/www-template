package persistence

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"www-template/packages/backend/internal/types"
)

const defaultInfrastructureTimeout = 3 * time.Second

func PingGormDatabase(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return errors.New("gorm database is required")
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("open sql db: %w", err)
	}

	pingContext, cancel := context.WithTimeout(ctx, defaultInfrastructureTimeout)
	defer cancel()
	if err := sqlDB.PingContext(pingContext); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}

	return nil
}

func (s *ValkeyStore) Ping(ctx context.Context) error {
	if s == nil || s.client == nil {
		return errors.New("valkey store is required")
	}

	pingContext, cancel := context.WithTimeout(ctx, defaultInfrastructureTimeout)
	defer cancel()
	if err := s.client.Ping(pingContext).Err(); err != nil {
		return fmt.Errorf("ping valkey: %w", err)
	}

	return nil
}

func CheckOpenSearch(ctx context.Context, config types.OpenSearchConfig) error {
	if strings.TrimSpace(config.URL) == "" {
		return errors.New("OPENSEARCH_URL is required")
	}

	requestContext, cancel := context.WithTimeout(ctx, defaultInfrastructureTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestContext, http.MethodGet, strings.TrimRight(config.URL, "/")+"/_cluster/health", nil)
	if err != nil {
		return fmt.Errorf("build opensearch request: %w", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("ping opensearch: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()
	_, _ = io.Copy(io.Discard, response.Body)
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("ping opensearch: unexpected status %d", response.StatusCode)
	}

	return nil
}

func CheckObjectStorage(ctx context.Context, config types.ObjectStorageConfig) error {
	if strings.TrimSpace(config.Endpoint) == "" {
		return errors.New("R2_ENDPOINT is required")
	}
	if strings.TrimSpace(config.Bucket) == "" {
		return errors.New("R2_BUCKET is required")
	}

	baseURL := strings.TrimRight(config.Endpoint, "/")
	bucketURL := baseURL + "/" + config.Bucket

	requestContext, cancel := context.WithTimeout(ctx, defaultInfrastructureTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestContext, http.MethodHead, bucketURL, nil)
	if err != nil {
		return fmt.Errorf("build object storage request: %w", err)
	}
	request.SetBasicAuth(config.AccessKeyID, config.SecretAccessKey)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("ping object storage: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()
	_, _ = io.Copy(io.Discard, response.Body)
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("ping object storage: unexpected status %d", response.StatusCode)
	}

	return nil
}
