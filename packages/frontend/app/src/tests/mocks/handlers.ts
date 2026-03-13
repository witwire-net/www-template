import { http, HttpResponse } from 'msw';

interface MockProfile {
  id: number;
  name: string;
  email: string;
  createdAt: string;
}

const initialProfiles: MockProfile[] = [
  {
    id: 1,
    name: 'Sample Profile 1',
    email: 'test1@example.com',
    createdAt: '2024-01-01T00:00:00.000Z',
  },
  {
    id: 2,
    name: 'Sample Profile 2',
    email: 'test2@example.com',
    createdAt: '2024-01-02T00:00:00.000Z',
  },
];

let profiles: MockProfile[] = [...initialProfiles];

const resetMockData = () => {
  profiles = [...initialProfiles];
};

/** MSW handlers for client-side API mocking. */
const handlers = [
  // GET /api/v1/profiles
  http.get('/api/v1/profiles', () => {
    return HttpResponse.json(profiles);
  }),

  // POST /api/v1/profiles
  http.post('/api/v1/profiles', async ({ request }) => {
    // Artificial delay so UI can show a loading state.
    await new Promise((resolve) => setTimeout(resolve, 75));

    const body = (await request.json()) as { name: string; email: string };
    const nextId = Math.max(0, ...profiles.map((u) => u.id)) + 1;
    const newProfile: MockProfile = {
      id: nextId,
      name: body.name,
      email: body.email,
      createdAt: new Date().toISOString(),
    };
    profiles = [...profiles, newProfile];
    return HttpResponse.json(newProfile, { status: 201 });
  }),

  // GET /api/v1/profiles/:id
  http.get('/api/v1/profiles/:id', ({ params }) => {
    const { id } = params;
    const profileId = Number(id);
    const found = profiles.find((u) => u.id === profileId);
    if (found == null) {
      return HttpResponse.json({ error: 'Not Found' }, { status: 404 });
    }
    return HttpResponse.json(found);
  }),
];

export { handlers, resetMockData };
