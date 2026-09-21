import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { chromium } from '@playwright/test';
import { expect, test } from '../helpers/fixtures';
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

    // SPEC gate 5.24 (no address bar, no tabs, no bookmarks bar) is the
    // browser's own doing for a window started this way, and a headless test
    // browser has no window chrome to inspect (it reports every bar as
    // visible), so it is not asserted here: what is asserted is the command
    // that produces it, and, in app-icon.spec.ts, the manifest that asks for
    // standalone. The bars themselves are owner check O5.3 and the real-window
    // screenshot taken at P5-08 (D-78).
    // It opens on the Briefing, which is where "/" goes (SPEC A8) and where an
    // installed copy's manifest points.
    expect(new URL(appPage.url()).pathname).toMatch(/^\/(briefing)?$/);
  } finally {
    await persistent.close();
    fs.rmSync(userDataDir, { recursive: true, force: true });
  }
});
