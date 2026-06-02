import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";

using Http;
using TypeSpec.OpenAPI;

namespace WWWTemplate;

@doc("Bearer client が context refresh で送る request body。Cookie mode では body を送らず、path-scoped HttpOnly Cookie だけを提示する。")
model BearerContextRefreshRequest {
  @doc("external bearer mode refresh であることを示す discriminator。")
  credentialMode: "bearer";

  @doc("rotation 対象の opaque refreshToken。Authorization header ではなく body だけで提示する。")
  refreshToken: string;
}

@doc("Product Cookie mode context refresh 成功時の response。refreshToken 平文を body に含めず、path-scoped Cookie rotation を transport header に委ねる。")
model ProductCookieContextRefreshResponse {
  ...CookieAuthEnvelope;

  @doc("認証された Product account の subject payload。")
  account: ProductAccountSubject;

  @doc("refresh token rotation 成功時点で Product Account から読み込んだ AccountSetting snapshot。")
  accountSetting?: AccountSettingSnapshot;
}

@doc("Product Bearer mode context refresh 成功時の response。Cookie command や browser context index hint を含めない。")
model ProductBearerContextRefreshResponse {
  ...BearerAuthEnvelope;

  @doc("認証された Product account の subject payload。")
  account: ProductAccountSubject;

  @doc("refresh token rotation 成功時点で Product Account から読み込んだ AccountSetting snapshot。")
  accountSetting?: AccountSettingSnapshot;
}

@doc("Product context refresh の credential mode 別 success response。Cookie mode は Cookie command を表現でき、Bearer mode は refreshToken body を返す。")
@oneOf
union RefreshTokenResponse {
  cookie: ProductCookieContextRefreshResponse,
  bearer: ProductBearerContextRefreshResponse,
}
