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
	"www-template/packages/backend/internal/platform/observability"
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

	requestStartedAt := time.Now()
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		wrappedErr := observability.WrapDatastoreConnectionError(fmt.Errorf("ping opensearch: %w", err))
		observeOpenSearchHealthCheck(ctx, requestStartedAt, 0, 0, wrappedErr)
		return wrappedErr
	}
	defer func() {
		_ = response.Body.Close()
	}()
	responseBytes, readErr := io.Copy(io.Discard, response.Body)
	if readErr != nil {
		wrappedErr := observability.WrapDatastoreConnectionError(fmt.Errorf("read opensearch health response: %w", readErr))
		observeOpenSearchHealthCheck(ctx, requestStartedAt, responseBytes, response.StatusCode, wrappedErr)
		return wrappedErr
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		wrappedErr := observability.WrapDatastoreUnexpectedStatusError(fmt.Errorf("ping opensearch: unexpected status %d", response.StatusCode))
		observeOpenSearchHealthCheck(ctx, requestStartedAt, responseBytes, response.StatusCode, wrappedErr)
		return wrappedErr
	}
	observeOpenSearchHealthCheck(ctx, requestStartedAt, responseBytes, response.StatusCode, nil)

	return nil
}

func observeOpenSearchHealthCheck(ctx context.Context, startedAt time.Time, responseBytes int64, statusCode int, err error) {
	// Step 1: health check endpoint は固定 path だけを target にし、cluster response body は属性化しない。
	status, errorClass := healthOpenSearchObservationStatus(err)
	statusCodeValue := int64(statusCode)

	// Step 2: response body はサイズだけを記録し、cluster 名や node 詳細が body から漏れないようにする。
	observability.ObserveDatastoreOperationCompleted(ctx, observability.DatastoreOperationCompleted{
		System:        observability.DatastoreSystemOpenSearch,
		Operation:     "health_check",
		Target:        "/_cluster/health",
		Status:        status,
		Duration:      time.Since(startedAt),
		ResponseBytes: healthPositiveInt64Pointer(responseBytes),
		StatusCode:    healthPositiveInt64Pointer(statusCodeValue),
		ResultClass:   observability.DatastoreResultClassStatus,
		ErrorClass:    errorClass,
	})
}

func healthOpenSearchObservationStatus(err error) (observability.DatastoreOperationStatus, observability.DatastoreErrorClass) {
	// Step 1: 成功時は status=ok/error_class=none として、成功 health check も同じ facet で検索できるようにする。
	if err == nil {
		return observability.DatastoreOperationStatusOK, observability.DatastoreErrorClassNone
	}

	// Step 2: caller cancellation は OpenSearch 側障害ではないため canceled として分離する。
	class := observability.ClassifyDatastoreError(err)
	if class == observability.DatastoreErrorClassCanceled {
		return observability.DatastoreOperationStatusCanceled, class
	}
	return observability.DatastoreOperationStatusError, class
}

func healthPositiveInt64Pointer(value int64) *int64 {
	// Step 1: 正の値だけを属性化し、未計測と 0 byte を混同しないようにする。
	if value <= 0 {
		return nil
	}
	return &value
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
