import { expect, test } from '../helpers/fixtures';
import { axeCheck } from '../helpers/axe';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { idFromHref, myPersonID, seedJob } from '../helpers/tasks';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

// SPEC gates 7.40-7.47 (B13.6, D-96): linked jobs on screen.

test.describe.configure({ timeout: 60_000 });

async function setup(page: Page, baseURL: string) {
  const me = await signInAsNewPerson(page, baseURL, '/tasks/board');
  const id = await myPersonID(page, baseURL);
  const tag = uniqueName('lnk').replace(/[^a-z0-9]/gi, '').toLowerCase();
  const titles = {
    news: `Print newsletter ${tag}`,
    toner: `Order toner ${tag}`,
    project: `Same project ${tag}`,
    gone: `Removed job ${tag}`,
  };
  for (const t of [titles.news, titles.toner, titles.project, titles.gone]) {
    await seedJob(page, baseURL, { title: t, stage: 'todo', personID: id });
  }
  const ids: Record<string, string> = {};
  await page.goto(baseURL + '/tasks/board?person=mine');
  await ready(page);
  for (const [key, title] of Object.entries(titles)) {
    ids[key] = idFromHref((await page.locator('.task-card', { hasText: title }).locator('.task-card-title a').first().getAttribute('href'))!);
  }
  return { me, id, tag, titles, ids };
}

const section = (page: Page) => page.locator('.task-links');
const card = (page: Page, title: string) => page.locator('.task-card', { hasText: title });

async function openWindow(page: Page, baseURL: string, title: string) {
  await page.goto(baseURL + '/tasks/board?person=mine');
  await ready(page);
  await card(page, title).locator('.task-card-title a').click();
  await expect(page.locator('#task-window[open]')).toBeVisible();
  await expect(section(page)).toBeVisible();
}

test('gates 7.40, 7.41: from the window, search, pick, choose the kind; both jobs show it in the right group', async ({ page, server }) => {
  const { titles, tag, ids } = await setup(page, server.baseURL);

  // Typing part of a title lists matches, never the job itself and never a removed job.
  await seedRemoved(page, server.baseURL, titles.gone);
  await openWindow(page, server.baseURL, titles.news);
  await section(page).getByLabel('Link a job').fill(tag);
  await expect(section(page).locator('.link-match-pick')).toHaveCount(2); // toner and project (not itself, not the removed one)
  await expect(section(page).locator('.link-results')).not.toContainText(titles.news);
  await expect(section(page).locator('.link-results')).not.toContainText(titles.gone);

  // Picking one asks what the link means.
  await section(page).locator('.link-match-pick', { hasText: titles.toner }).click();
  await expect(section(page).getByRole('button', { name: 'That one first' })).toBeVisible();
  await expect(section(page).getByRole('button', { name: 'This one first' })).toBeVisible();
  await expect(section(page).getByRole('button', { name: 'Related' })).toBeVisible();
  await axeCheck(page);
  await section(page).getByRole('button', { name: 'That one first' }).click();
  await expect(section(page).locator('.link-first .link-line', { hasText: titles.toner })).toHaveCount(1);
  await expect(section(page).locator('.link-first .link-line', { hasText: 'To do' })).toHaveCount(1);

  // Related, then "This one first", the same way.
  await section(page).getByLabel('Link a job').fill(titles.project);
  await section(page).locator('.link-match-pick', { hasText: titles.project }).click();
  await section(page).getByRole('button', { name: 'Related' }).click();
  await expect(section(page).locator('.link-related .link-line', { hasText: titles.project })).toHaveCount(1);

  // The other job shows the same link from its own side (here from its own page).
  await page.goto(server.baseURL + `/tasks/${ids.toner}`);
  await ready(page);
  await expect(section(page).locator('.link-then .link-line', { hasText: titles.news })).toHaveCount(1);
});

async function seedRemoved(page: Page, baseURL: string, title: string): Promise<string> {
  await page.goto(baseURL + '/tasks/board?person=mine');
  await ready(page);
  const id = idFromHref((await card(page, title).locator('.task-card-title a').first().getAttribute('href'))!);
  const res = await page.request.post(`${baseURL}/tasks/${id}/remove`, { headers: { origin: baseURL }, maxRedirects: 0 });
  expect(res.status()).toBeLessThan(400);
  return id;
}

test('gates 7.42, 7.43: WAITING on the Board, My jobs and Team, naming what it waits on; clears when the first job is Done; comes back on Reopen; never blocks', async ({ page, server }) => {
  const { titles, ids } = await setup(page, server.baseURL);
  await page.request.post(`${server.baseURL}/tasks/${ids.news}/links`, {
    form: { other_id: ids.toner, kind: 'first' },
    headers: { origin: server.baseURL },
    maxRedirects: 0,
  });
  const text = `Waiting on: ${titles.toner}`;

  // Board
  await page.goto(server.baseURL + '/tasks/board?person=mine');
  await ready(page);
  const stamp = card(page, titles.news).locator('.stamp.waiting');
  await expect(stamp).toHaveText('WAITING');
  await expect(stamp).toHaveAttribute('title', text);
  await expect(stamp).toHaveAttribute('aria-label', text);
  await expect(card(page, titles.toner).locator('.stamp.waiting')).toHaveCount(0); // the one it waits on isn't waiting

  // My jobs and Team
  await page.goto(server.baseURL + '/tasks');
  await ready(page);
  await expect(page.locator('.myjobs-upnext-list .team-task', { hasText: titles.news }).locator('.stamp.waiting')).toHaveAttribute('aria-label', text);
  await page.goto(server.baseURL + '/tasks/team');
  await ready(page);
  await expect(page.locator('.team-lane.me .team-task', { hasText: titles.news }).locator('.stamp.waiting')).toHaveAttribute('aria-label', text);

  // Waiting is only a label: the waiting job can still be started and finished.
  await page.goto(server.baseURL + '/tasks');
  await ready(page);
  await page.locator('.myjobs-upnext-list .team-task', { hasText: titles.news }).getByRole('button', { name: 'Start' }).click();
  await ready(page);
  await expect(page.locator('.myjobs-working-list .team-task', { hasText: titles.news })).toHaveCount(1);
  await expect(page.locator('.myjobs-working-list .team-task', { hasText: titles.news }).locator('.stamp.waiting')).toHaveCount(1);

  // Finishing the first job clears the stamp; reopening brings it back.
  await page.locator('.myjobs-upnext-list .team-task', { hasText: titles.toner }).getByRole('button', { name: 'Done ✓' }).click();
  await ready(page);
  await expect(page.locator('.myjobs-working-list .team-task', { hasText: titles.news }).locator('.stamp.waiting')).toHaveCount(0);
  await page.goto(server.baseURL + '/tasks/board?person=mine');
  await ready(page);
  await expect(card(page, titles.news).locator('.stamp.waiting')).toHaveCount(0);
  const reopen = await page.request.post(`${server.baseURL}/tasks/${ids.toner}/reopen`, { headers: { origin: server.baseURL }, maxRedirects: 0 });
  expect(reopen.status()).toBeLessThan(400);
  await page.reload();
  await ready(page);
  await expect(card(page, titles.news).locator('.stamp.waiting')).toHaveAttribute('aria-label', text);

  // And the waiting job can be finished while it is still waiting.
  await page.goto(server.baseURL + '/tasks');
  await ready(page);
  await page.locator('.myjobs-working-list .team-task', { hasText: titles.news }).getByRole('button', { name: 'Done ✓' }).click();
  await ready(page);
  await page.goto(server.baseURL + '/tasks/board?person=mine');
  await ready(page);
  await expect(page.locator('.task-column[data-stage="done"] .task-card', { hasText: titles.news })).toHaveCount(1);
});

test('gates 7.44, 7.46: the remove icon takes the link from both jobs, with History; a removed job hides its links, and Undo brings them back', async ({ page, server }) => {
  const { me, titles, ids } = await setup(page, server.baseURL);
  await page.request.post(`${server.baseURL}/tasks/${ids.news}/links`, {
    form: { other_id: ids.toner, kind: 'first' },
    headers: { origin: server.baseURL },
    maxRedirects: 0,
  });

  // Removing and Undoing the other job keeps the link.
  await page.goto(server.baseURL + '/tasks/board?person=mine');
  await ready(page);
  await card(page, titles.toner).getByRole('button', { name: `Remove ${titles.toner}` }).click();
  await expect(page.locator('#toast-region .toast')).toHaveCount(1);
  await page.goto(server.baseURL + `/tasks/${ids.news}`);
  await ready(page);
  await expect(section(page).locator('.link-line')).toHaveCount(0); // hidden while it is removed
  await page.goBack();
  await ready(page);
  await page.goto(server.baseURL + '/tasks/board?person=mine');
  await ready(page);

  // Restore from Removed tasks brings it back (7.46) ...
  await page.goto(server.baseURL + '/tasks/removed');
  await ready(page);
  await page.locator('.task-simple-list li', { hasText: titles.toner }).getByRole('button', { name: 'Restore' }).click();
  await page.goto(server.baseURL + `/tasks/${ids.news}`);
  await ready(page);
  await expect(section(page).locator('.link-first .link-line', { hasText: titles.toner })).toHaveCount(1);

  // The remove icon, from the page: both jobs lose it, History on both.
  await section(page).getByRole('button', { name: `Remove link to ${titles.toner}` }).click();
  await expect(section(page).locator('.link-line')).toHaveCount(0);
  await page.goto(server.baseURL + `/tasks/${ids.toner}`);
  await ready(page);
  await expect(section(page).locator('.link-line')).toHaveCount(0);
  await expect(page.locator('.task-activity-list')).toContainText(`${me} removed the link to “${titles.news}” (then)`);
  await page.goto(server.baseURL + `/tasks/${ids.news}`);
  await ready(page);
  await expect(page.locator('.task-activity-list')).toContainText(`${me} linked this to “${titles.toner}” (do first)`);
  await expect(page.locator('.task-activity-list')).toContainText(`${me} removed the link to “${titles.toner}” (do first)`);
});

test('gate 7.16 with links: Undo after removing a job keeps its links', async ({ page, server }) => {
  const { titles, ids } = await setup(page, server.baseURL);
  await page.request.post(`${server.baseURL}/tasks/${ids.news}/links`, {
    form: { other_id: ids.toner, kind: 'related' },
    headers: { origin: server.baseURL },
    maxRedirects: 0,
  });
  await page.goto(server.baseURL + '/tasks/board?person=mine');
  await ready(page);
  await card(page, titles.news).getByRole('button', { name: `Remove ${titles.news}` }).click();
  await page.locator('#toast-region .toast').getByRole('button', { name: 'Undo' }).click();
  await expect(page.locator('#toast-region .toast')).toHaveCount(0);
  await page.goto(server.baseURL + `/tasks/${ids.news}`);
  await ready(page);
  await expect(section(page).locator('.link-related .link-line', { hasText: titles.toner })).toHaveCount(1);
});

test('gate 7.45: the refusals are plain messages on the page, and change nothing', async ({ page, server }) => {
  const { titles, ids } = await setup(page, server.baseURL);
  const post = async (task: string, other: string, kind: string) => {
    const res = await page.request.post(`${server.baseURL}/tasks/${task}/links`, {
      form: { other_id: other, kind },
      headers: { origin: server.baseURL },
      maxRedirects: 0,
    });
    return { status: res.status(), body: await res.text() };
  };
  expect((await post(ids.news, ids.toner, 'first')).status).toBeLessThan(400);
  expect((await post(ids.news, ids.toner, 'related')).body).toContain('Those two jobs are already linked.');
  expect((await post(ids.news, ids.news, 'related')).body).toContain("A job can&#39;t be linked to itself.");
  // Print newsletter waits for Order toner; Order toner can't be made to wait for it by way of a third.
  expect((await post(ids.toner, ids.project, 'first')).status).toBeLessThan(400); // Same project first, then Order toner
  const loop = await post(ids.project, ids.news, 'first'); // Print newsletter first, then Same project: circle
  expect(loop.body).toContain('already has to wait for this job');
  // A job somebody has just removed.
  const goneId = await seedRemoved(page, server.baseURL, titles.gone);
  expect((await post(ids.news, goneId, 'related')).body).toContain('has just been removed');
});

test.describe('with JavaScript switched off', () => {
  test.use({ javaScriptEnabled: false });

  test('gate 7.47: linking and unlinking work from the job\'s own page as ordinary forms', async ({ page, server }) => {
    const { titles, ids, tag } = await setup(page, server.baseURL);
    await page.goto(server.baseURL + `/tasks/${ids.news}`);
    await section(page).getByLabel('Link a job').fill(tag);
    await section(page).getByRole('button', { name: 'Find' }).click();
    await expect(section(page).locator('.link-match')).toHaveCount(3); // toner, project and the removed-in-a-moment one, but not itself
    await expect(section(page).locator('.link-results')).not.toContainText(titles.news);
    await section(page).locator('.link-match', { hasText: titles.toner }).getByRole('button', { name: 'This one first' }).click();
    await expect(section(page).locator('.link-then .link-line', { hasText: titles.toner })).toHaveCount(1);
    await section(page).getByRole('button', { name: `Remove link to ${titles.toner}` }).click();
    await expect(section(page).locator('.link-line')).toHaveCount(0);
  });
});
