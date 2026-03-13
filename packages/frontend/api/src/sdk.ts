import {
  createProfile,
  getStatus,
  getProfile,
  listProfiles,
  type CreateProfileInput,
  type getStatusResponse,
} from './generated/client.js';

export type {
  CreateProfileInput,
  StatusResponse,
  Profile,
  ErrorResponse,
  createProfileResponse,
  getStatusResponse,
  getProfileResponse,
  listProfilesResponse,
} from './generated/client.js';

/** Configuration for the API SDK default request settings. */
/** Configuration for the API SDK default request settings. */
interface ApiSdkConfig {
  defaultInit?: RequestInit;
}

const toHeaderObject = (headers?: HeadersInit): Record<string, string> => {
  if (headers == null) {
    return {};
  }
  // Normalize using the built-in Headers implementation to avoid unsafe casts
  const normalized = new Headers(headers);
  return Object.fromEntries(normalized.entries());
};

const withDefaultInit = (init: RequestInit | undefined, defaultInit: RequestInit | undefined) => {
  if (defaultInit == null) {
    return init;
  }
  return {
    ...defaultInit,
    ...init,
    headers: {
      ...toHeaderObject(defaultInit.headers),
      ...toHeaderObject(init?.headers),
    },
  };
};

/** Create a typed API SDK wrapper with optional default request init. */
/** Create a typed API SDK wrapper with optional default request init. */
const createApiSdk = (config?: ApiSdkConfig) => {
  const defaultInit = config?.defaultInit;

  return {
    status: {
      get: (options?: RequestInit): Promise<getStatusResponse> =>
        getStatus(withDefaultInit(options, defaultInit)),
    },
    profiles: {
      list: (options?: RequestInit) => listProfiles(withDefaultInit(options, defaultInit)),
      create: (payload: CreateProfileInput, options?: RequestInit) =>
        createProfile(
          payload,
          withDefaultInit(
            {
              ...options,
              headers: {
                'Content-Type': 'application/json',
                ...toHeaderObject(options?.headers),
              },
            },
            defaultInit
          )
        ),
      get: (id: number, options?: RequestInit) =>
        getProfile(id, withDefaultInit(options, defaultInit)),
    },
  };
};

export type { ApiSdkConfig };
export { createApiSdk };
