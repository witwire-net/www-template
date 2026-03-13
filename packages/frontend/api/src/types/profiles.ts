/** Profile record mapped for client consumption. */
interface Profile {
  id: number;
  name: string;
  email: string;
  createdAt: Date;
}

/** Payload to create a new profile. */
interface CreateProfilePayload {
  name: string;
  email: string;
}

export type { CreateProfilePayload, Profile };
