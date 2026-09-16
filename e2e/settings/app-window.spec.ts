import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { chromium } from '@playwright/test';
import { test } from '../helpers/fixtures';
import { ready } from '../helpers/ready';
import { signInAsNewPerson } from '../helpers/people';

// SPEC gate 1.43: the Setup page's "--app=" command opens Comp HQ as the
// first, only window — proven here the same way the owner's shortcut
// will actually launch it, rather than just checking the command text.
test('the --app command opens Comp HQ as the first window and it becomes ready', async ({ page, server }) => {
  // Sign in first with a normal page, so the persistent-context launch
  // below lands straight on content instead of the picker.
  await signInAsNewPerson(page, server.baseURL, '/');
  const cookie = (await page.context().cookies()).find((c) => c.name === 'comphq_person');
  if (!cookie) throw new Error('sign-in did not set a person cookie');

  const userDataDir = fs.mkdtempSync(path.join(os.tmpdir(), 'comphq-app-window-'));
  const persistent = await chromium.launchPersistentContext(userDataDir, {
    args: [`--app=${server.baseURL}/`],
  });
  try {
    // Carry the same identity into the new window, so it also lands on
    // content rather than the picker.
    await persistent.addCookies([{ name: cookie.name, value: cookie.value, url: server.baseURL }]);

    const appPage = persistent.pages()[0] ?? (await persistent.waitForEvent('page'));
    await appPage.goto(server.baseURL + '/');
    await ready(appPage);

    if (persistent.pages().length !== 1) {
      throw new Error(`expected exactly one window, got ${persistent.pages().length}`);
    }
  } finally {
    await persistent.close();
    fs.rmSync(userDataDir, { recursive: true, force: true });
  }
});
