import { http, HttpResponse } from 'msw';

const resetMockData = () => undefined;

/** MSW handlers for client-side API mocking. */
const handlers = [
  // GET /api/v1/status
  http.get('/api/v1/status', () => {
    return HttpResponse.json(
      {
        status: 'ok',
      },
      {
        status: 200,
      }
    );
  }),
];

export { handlers, resetMockData };
