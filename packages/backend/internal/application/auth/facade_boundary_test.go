package auth

import (
	"embed"
	"strings"
	"testing"
)

// authApplicationSourceFiles は Auth application の production source を test binary に固定して埋め込む。
// 実行時 path を受け取らないことで、source boundary guard が任意 file 読み込みにならないことを保証する。
//
//go:embed account_facade_contracts.go product_facade_contracts.go facade_service.go
var authApplicationSourceFiles embed.FS

// TestAuthApplicationSourceBoundary は Auth application が AccountSetting や Account 代替モデルを所有しないことを検証する。
func TestAuthApplicationSourceBoundary(t *testing.T) {
	t.Parallel()

	// [LOCALIZATION-BE-S014] ARCH-BE-ACCOUNT-AUTH-SUBORDINATION / ARCH-BE-AUTH-NO-ACCOUNT-SETTING は production source だけを対象にする。
	for _, filePath := range fixedAuthBoundaryFiles() {
		content, err := authApplicationSourceFiles.ReadFile(filePath)
		if err != nil {
			t.Fatalf("read auth boundary source %s: %v", filePath, err)
		}
		assertAuthBoundaryClean(t, filePath, stripGoComments(string(content)))
	}
}

func fixedAuthBoundaryFiles() []string {
	// gosec G304 を避けるため、検査対象は Auth domain/application の固定 production file に限定する。
	return []string{
		"account_facade_contracts.go",
		"product_facade_contracts.go",
		"facade_service.go",
	}
}

func assertAuthBoundaryClean(t *testing.T, filePath string, source string) {
	t.Helper()

	// Auth source は Account.Auth projection の語彙だけを持ち、Product AccountSetting や旧認証アカウント語彙を持たない。
	forbiddenTerms := []string{"Auth" + "Account", "Auth" + "Subject", "Auth" + "AccountRepository", "AccountClient" + "Settings", "AccountSetting", "AccountLocale"}
	for _, term := range forbiddenTerms {
		if strings.Contains(source, term) {
			t.Fatalf("%s must not contain %s in Auth production source", filePath, term)
		}
	}
}

func stripGoComments(source string) string {
	// boundary guard は実装上の所有を検査するため、設計意図を説明するコメントは除外して誤検知を避ける。
	withoutBlockComments := stripDelimited(source, "/*", "*/")
	lines := strings.Split(withoutBlockComments, "\n")
	for index, line := range lines {
		if position := strings.Index(line, "//"); position >= 0 {
			lines[index] = line[:position]
		}
	}
	return strings.Join(lines, "\n")
}

func stripDelimited(source string, start string, end string) string {
	// block comment は複数行にまたがるため、開始・終了 marker の範囲を繰り返し削除する。
	for {
		startIndex := strings.Index(source, start)
		if startIndex < 0 {
			return source
		}
		endIndex := strings.Index(source[startIndex+len(start):], end)
		if endIndex < 0 {
			return source[:startIndex]
		}
		removeEnd := startIndex + len(start) + endIndex + len(end)
		source = source[:startIndex] + source[removeEnd:]
	}
}
