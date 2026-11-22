import { expect, test } from '../helpers/fixtures';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';

// Questions across an upgrade and a rollback (SPEC gate 8.82, B14.6, D-110): two
// questions are asked under the new build, one with an answer and marked answered;
// the container is rolled back to the previous release, which has never heard of
// Questions and must carry on as if the section weren't there; upgraded again,
// every question, answer and mark is still there.
//
// Tagged @questions-seed / @questions-rolledback / @questions-verify-upgraded (none a
// substring of another) for container-test.sh's phases. The texts are fixed because
// a later phase has to find them; the person is not.
const WAITING = 'Smoke question that is still waiting';
const ANSWERED = 'Smoke question that was answered';
const ANSWER = 'Smoke answer to the answered one';

type Page = import('@playwright/test').Page;

function form(pairs: [string, string][]): string {
  return new URLSearchParams(pairs).toString();
}

test('@questions-seed two questions, one answered and marked answered', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/questions');
  const post = async (path: string, pairs: [string, string][]) => {
    const res = await page.request.post(server.baseURL + path, {
      data: form(pairs),
      headers: { origin: server.baseURL, 'content-type': 'application/x-www-form-urlencoded' },
      maxRedirects: 0,
    });
    expect(res.status(), path).toBeLessThan(400);
  };
  await post('/questions', [['text', WAITING], ['details', 'Some details']]);
  await post('/questions', [['text', ANSWERED]]);
  await page.goto(server.baseURL + '/questions');
  await ready(page);
  const href = await page.locator('.q-row', { hasText: ANSWERED }).locator('a').first().getAttribute('href');
  const id = /\/questions\/(\d+)/.exec(href!)![1];
  await post(`/questions/${id}/answers`, [['text', ANSWER]]);
  await post(`/questions/${id}/answered`, []);
});

async function assertAllThere(page: Page, baseURL: string) {
  await page.goto(baseURL + '/questions');
  await ready(page);
  await expect(page.locator('section[aria-labelledby="q-waiting-heading"] .q-row', { hasText: WAITING })).toHaveCount(1);
  const answered = page.locator('section[aria-labelledby="q-answered-heading"] .q-row', { hasText: ANSWERED });
  await expect(answered).toHaveCount(1);
  await expect(answered.locator('.q-row-meta')).toContainText('1 answer');
  await answered.locator('a').click();
  await ready(page);
  await expect(page.locator('.q-answer')).toHaveText(new RegExp(ANSWER));
  await expect(page.locator('.q-answered-line')).toContainText('Answered · marked by');
}

test('@questions-readback the questions are there on the build that made them', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/questions');
  await assertAllThere(page, server.baseURL);
});

// "1.5.0" >= "1.6.0": compares dotted release numbers part by part.
function atLeast(version: string, min: string): boolean {
  const a = version.split('.').map(Number);
  const b = min.split('.').map(Number);
  for (let i = 0; i < Math.max(a.length, b.length); i++) {
    const d = (a[i] ?? 0) - (b[i] ?? 0);
    if (d !== 0) return d > 0;
  }
  return true;
}

// Run against the PREVIOUS release's app. container-test.sh passes its version as
// COMPHQ_PREV_VERSION. A release before 1.6.0 has no Questions: its sidebar doesn't
// show the section, the address is just "Page not found", and everything else still
// works. A later release still has them.
test('@questions-rolledback the previous release carries on as if there were no Questions', async ({ page, server }) => {
  const prev = process.env.COMPHQ_PREV_VERSION ?? '';
  const prevKnows = prev === '' || atLeast(prev, '1.6.0');
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);
  if (prevKnows) {
    await assertAllThere(page, server.baseURL);
    return;
  }
  await expect(page.locator('.sidebar nav a[href="/questions"]')).toHaveCount(0);
  const res = await page.request.get(server.baseURL + '/questions');
  expect(res.status()).toBe(404);
  await page.goto(server.baseURL + '/briefing');
  await ready(page);
  await expect(page.getByText('Must be done')).toBeVisible();
});

test('@questions-verify-upgraded every question, answer and mark is still there after the upgrade', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/questions');
  await assertAllThere(page, server.baseURL);
});
