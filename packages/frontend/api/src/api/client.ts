import { createApiSdk, type StatusResponse, type Profile } from '../sdk';

import type { CreateProfilePayload, Status } from '../types';

const sdk = createApiSdk();

const toStatus = (dto: StatusResponse): Status => ({
  message: dto.message,
  timestamp: new Date(dto.timestamp),
});

const toProfile = (dto: Profile) => ({
  id: dto.id,
  name: dto.name,
  email: dto.email,
  createdAt: new Date(dto.createdAt),
});

/** Status API wrapper for the public sample endpoint. */
const statusApi = {
  get: async (): Promise<Status> => {
    const { data } = await sdk.status.get();
    return toStatus(data);
  },
};

/** Profiles API wrapper for list/create/get operations. */
const profilesApi = {
  list: async () => {
    const response = (await sdk.profiles.list()) as { data: unknown; status: number };
    if (response.status !== 200) {
      const maybeError = response.data as { error?: string };
      throw new Error(maybeError.error ?? 'Failed to fetch profiles');
    }
    if (!Array.isArray(response.data)) {
      throw new TypeError('Invalid profiles response');
    }
    return response.data.map((user) => toProfile(user as Profile));
  },
  create: async (payload: CreateProfilePayload) => {
    const response = await sdk.profiles.create(payload);
    if (response.status !== 201) {
      const maybeError = response.data as { error?: string };
      throw new Error(maybeError.error ?? 'Failed to create profile');
    }
    return toProfile(response.data);
  },
  get: async (id: number) => {
    const response = await sdk.profiles.get(id);
    if (response.status !== 200) {
      return null;
    }
    return toProfile(response.data);
  },
};

export { statusApi, profilesApi };

// SDK types are internal; consumers should use domain types
