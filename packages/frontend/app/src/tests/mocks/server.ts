import { setupServer } from 'msw/node';

import { handlers } from './handlers';

/** MSW server instance for client test requests. */
const server = setupServer(...handlers);

export { server };
