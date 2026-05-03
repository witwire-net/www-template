package health

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"www-template/packages/backend/internal/platform/config"
)

const defaultInfrastructureTimeout = 3 * time.Second

func CheckOpenSearch(ctx context.Context, cfg config.OpenSearchConfig) error {
	if strings.TrimSpace(cfg.URL) == "" {
		return errors.New("OPENSEARCH_URL is required")
	}

	requestContext, cancel := context.WithTimeout(ctx, defaultInfrastructureTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestContext, http.MethodGet, strings.TrimRight(cfg.URL, "/")+"/_cluster/health", nil)
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

func CheckObjectStorage(ctx context.Context, cfg config.ObjectStorageConfig) error {
	if strings.TrimSpace(cfg.Endpoint) == "" {
		return errors.New("R2_ENDPOINT is required")
	}
	if strings.TrimSpace(cfg.Bucket) == "" {
		return errors.New("R2_BUCKET is required")
	}

	baseURL := strings.TrimRight(cfg.Endpoint, "/")
	bucketURL := baseURL + "/" + cfg.Bucket

	requestContext, cancel := context.WithTimeout(ctx, defaultInfrastructureTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestContext, http.MethodHead, bucketURL, nil)
	if err != nil {
		return fmt.Errorf("build object storage request: %w", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("ping object storage: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()
	_, _ = io.Copy(io.Discard, response.Body)
	// 200-299: accessible. 403: reachable but auth required (expected for S3/MinIO without signing).
	// Any other status indicates a configuration or connectivity problem.
	if response.StatusCode == http.StatusForbidden {
		return nil
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("ping object storage: unexpected status %d", response.StatusCode)
	}

	return nil
}
