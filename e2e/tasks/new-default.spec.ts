import { expect, test } from '../helpers/fixtures';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { openNewTask } from '../helpers/tasks';
import { uniqueName } from '../helpers/unique-name';

// SPEC gates 7.10 and 7.11 (D-90): a new job starts in To do, and a new job
// with people on it is in their Up next straight away.

test('gate 7.10: the add-a-task window and page open with To do already chosen', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  await page.click('#add-task-link');
  await expect(page.locator('#new-task-stage')).toHaveValue('todo');
  await expect(page.locator('#new-task-stage option:checked')).toHaveText('To do');

  await openNewTask(page);
  await expect(page.locator('#new-task-stage')).toHaveValue('todo');
});

test('gate 7.10: a job made with no column chosen goes to the bottom of To do', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  const first = uniqueName('First job');
  const second = uniqueName('Second job');
  const todo = page.locator('.task-column[data-stage="todo"] .task-card');
  const before = await todo.locator('.task-card-title').allInnerTexts();
  for (const title of [first, second]) {
    await openNewTask(page);
    await page.fill('#new-task-title', title);
    await page.click('.add-task-form button[type="submit"]');
    await ready(page);
  }
  // The server is shared with other tests, which add jobs of their own at the bottom too, so only
  // the jobs that were there before and the two made here are compared: the two come last, in order.
  const titles = (await todo.locator('.task-card-title').allInnerTexts()).filter(
    (t) => before.includes(t) || t.includes(first) || t.includes(second),
  );
  expect(titles.slice(-2).map((t) => (t.includes(first) ? 'first' : t.includes(second) ? 'second' : t))).toEqual(['first', 'second']);
  expect(titles.slice(0, -2)).toEqual(before.filter((t) => titles.includes(t)));
  await expect(page.locator('.task-column[data-stage="idea"] .task-card', { hasText: first })).toHaveCount(0);

  // An idea can still be chosen.
  const idea = uniqueName('An idea');
  await openNewTask(page);
  await page.fill('#new-task-title', idea);
  await page.selectOption('#new-task-stage', 'idea');
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
  await expect(page.locator('.task-column[data-stage="idea"] .task-card', { hasText: idea })).toHaveCount(1);
});

test('gate 7.11: a new job with a person on it is in their Up next and counts towards their workload', async ({ page, server }) => {
  const me = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  const title = uniqueName('Order toner');

  await openNewTask(page);
  await page.fill('#new-task-title', title);
  await page.selectOption('#new-task-people', { label: me });
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);

  await page.goto(server.baseURL + '/tasks');
  await ready(page);
  await expect(page.locator('.myjobs-upnext-list .team-task', { hasText: title })).toHaveCount(1);
  // A medium job is 2 workload blocks.
  await expect(page.locator('.myjobs-mine .workload-block')).toHaveCount(2);

  await page.goto(server.baseURL + '/tasks/team');
  await ready(page);
  const lane = page.locator('.team-lane.me');
  await expect(lane.locator('.team-up-next-list .team-task', { hasText: title })).toHaveCount(1);
  await expect(lane.locator('.workload-block')).toHaveCount(2);
});
