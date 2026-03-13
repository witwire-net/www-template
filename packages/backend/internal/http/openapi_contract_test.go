package http

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

type openAPIContract struct {
	Paths map[string]struct {
		Get struct {
			Security []map[string][]string `json:"security"`
		} `json:"get"`
	} `json:"paths"`
	Components struct {
		SecuritySchemes map[string]struct {
			Scheme string `json:"scheme"`
			Type   string `json:"type"`
		} `json:"securitySchemes"`
	} `json:"components"`
}

func TestAppOpenAPIDeclaresBearerSecurity(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("../../../typespec/openapi/openapi.json")
	if err != nil {
		t.Fatalf("read openapi: %v", err)
	}

	var contract openAPIContract
	if err := json.Unmarshal(content, &contract); err != nil {
		t.Fatalf("unmarshal openapi: %v", err)
	}

	bearerAuth, ok := contract.Components.SecuritySchemes["BearerAuth"]
	if !ok {
		t.Fatal("BearerAuth security scheme not found")
	}
	if bearerAuth.Type != "http" || strings.ToLower(bearerAuth.Scheme) != "bearer" {
		t.Fatalf("unexpected BearerAuth scheme: type=%s scheme=%s", bearerAuth.Type, bearerAuth.Scheme)
	}

	assertBearerSecurity(t, contract, "/api/v1/app/profiles")
	assertBearerSecurity(t, contract, "/api/v1/app/profiles/{id}")
}

func assertBearerSecurity(t *testing.T, contract openAPIContract, path string) {
	t.Helper()

	item, ok := contract.Paths[path]
	if !ok {
		t.Fatalf("path %s not found", path)
	}

	for _, entry := range item.Get.Security {
		if _, ok := entry["BearerAuth"]; ok {
			return
		}
	}

	t.Fatalf("BearerAuth security missing for %s", path)
}
