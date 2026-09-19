import { test as base } from '../helpers/fixtures';
import { Mockup } from './mockup';

// Every design test gets `mockup`: docs/design/mockup.html, open in a
// second tab of the same browser as the app's own page.
export const test = base.extend<{ mockup: Mockup }>({
  mockup: async ({ context }, use) => {
    const mockup = await Mockup.open(context);
    await use(mockup);
    await mockup.close();
  },
});

export { expect } from '@playwright/test';
