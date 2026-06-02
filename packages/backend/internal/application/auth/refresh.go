package auth

import domain "www-template/packages/backend/internal/domain"

// RefreshCredentialHash は canonical auth lifecycle が store 境界へ渡す refresh credential hash DTO である。
//
// 役割:
//   - application/auth の公開 API が domain 型を直接露出しないよう、保存・照合用 hash の application 境界名を提供する。
//   - 実体は domain.OpaqueTokenHash と同じ値であり、既存 store port へ追加変換なしに渡せる。
//
// 使用例:
//
//	hash, err := auth.HashRefreshCredential(refreshToken)
//	if err != nil {
//		return err
//	}
type RefreshCredentialHash = domain.OpaqueTokenHash

// HashRefreshCredential は refresh credential の平文値を保存・照合用 hash に変換する。
//
// 役割:
//   - Product/Admin の refresh family use case が同じ opaque token hash primitive を使う入口を提供する。
//   - 平文 refreshToken を永続化せず、domain.OpaqueTokenHash へ変換してから store 境界へ渡せるようにする。
//   - account/operator の eligibility や service artifact 判断を持たず、credential primitive だけを扱う。
//
// 引数:
//   - refreshToken: Cookie または request body で提示された opaque refresh credential 平文。
//
// 戻り値:
//   - domain.OpaqueTokenHash: 保存・照合に使う digest value。
//   - error: 空 token など、hash primitive が拒否した場合の domain error。
//
// 使用例:
//
//	hash, err := auth.HashRefreshCredential(refreshToken)
//	if err != nil {
//		return err
//	}
func HashRefreshCredential(refreshToken string) (RefreshCredentialHash, error) {
	// Step 1: hash algorithm と空値検証は domain primitive に直接委譲し、application shared wrapper を残さない。
	hash, err := domain.HashOpaqueToken(refreshToken)
	if err != nil {
		return "", err
	}

	// Step 2: concept lifecycle の store port が扱う domain hash value に変換し、平文 token は返さない。
	return hash, nil
}
