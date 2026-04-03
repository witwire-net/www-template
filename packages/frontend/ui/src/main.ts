import { mount } from 'svelte';

import App from './App.svelte';
import './styles/index';

const rootElement = document.getElementById('root');

if (rootElement === null) {
  throw new Error('Root element not found.');
}

mount(App, {
  target: rootElement,
});
