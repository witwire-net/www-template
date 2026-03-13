import "@typespec/openapi";

using TypeSpec.OpenAPI;

namespace WwwTemplate;

model Profile {
  id: int64;
  name: string;
  @format("email") email: string;
  createdAt: utcDateTime;
}

model CreateProfileInput {
  name: string;
  @format("email") email: string;
}
