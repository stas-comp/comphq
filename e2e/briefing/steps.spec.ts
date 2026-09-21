import { expect, test } from '../helpers/fixtures-today';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { openNewTask } from '../helpers/tasks';

// D-73: the Saturday Briefing does not read steps. A job with unfinished
// steps briefs exactly as it does without any (PLAN P5-04). Its own server,
// with a fixed "today" (SPEC B4), like the rest of the Briefing tests.
test.use({ today: '2026-09-19' });

test('@fresh D-73: adding and ticking steps changes nothing on the Briefing', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);
  await openNewTask(page);
  await page.fill('#new-task-title', 'Concert night');
  await page.selectOption('#new-task-stage', 'todo');
  await page.fill('#new-task-due-date', '2026-09-23');
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
  const href = await page.locator('.task-card', { hasText: 'Concert night' }).locator('.task-card-title a').first().getAttribute('href');
  const id = href!.split('/').pop()!;

  const briefing = async () => {
    await page.goto(server.baseURL + '/briefing');
    await ready(page);
    return page.locator('main').innerHTML();
  };
  const before = await briefing();
  expect(before).toContain('Concert night'); // it is briefed, so the comparison means something

  const post = async (p: string, form: Record<string, string>) => {
    const res = await page.request.post(server.baseURL + p, { headers: { origin: server.baseURL }, form, maxRedirects: 0 });
    expect([200, 302]).toContain(res.status());
  };
  for (const text of ['Print exam papers', 'Book the hall', 'Email the parents']) await post(`/tasks/${id}/steps`, { text });
  await page.goto(`${server.baseURL}/tasks/${id}`);
  const sid = await page.locator('.task-steps .step').first().getAttribute('data-step-id');
  await post(`/tasks/${id}/steps/${sid}/tick`, { done: '1' });

  expect(await briefing(), 'a job with unfinished steps briefs exactly as it did without them').toBe(before);
  expect(await page.locator('main').innerText()).not.toMatch(/steps? done|Print exam papers|\d\/\d/);
});
