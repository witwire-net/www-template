package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"www-template/packages/backend/internal/platform/config"
)

func TestCheckOpenSearchRejectsUnexpectedStatus(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	err := CheckOpenSearch(context.Background(), config.OpenSearchConfig{URL: server.URL})
	if err == nil {
		t.Fatal("expected opensearch health check failure")
	}
}

func TestCheckObjectStorageAcceptsBucketHead(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/template" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := CheckObjectStorage(context.Background(), config.ObjectStorageConfig{
		Endpoint:        server.URL,
		Bucket:          "template",
		Region:          "us-east-1",
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin",
		UsePathStyle:    true,
	})
	if err != nil {
		t.Fatalf("expected object storage health check success, got %v", err)
	}
}
