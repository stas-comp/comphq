import { expect, test } from '../helpers/fixtures';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { uniqueName } from '../helpers/unique-name';
import { openNewTask, openTaskPage } from '../helpers/tasks';

type Page = import('@playwright/test').Page;

async function addTask(page: Page, title: string, stage: string, personName?: string, size?: string): Promise<void> {
  await openNewTask(page);
  await page.fill('#new-task-title', title);
  await page.selectOption('#new-task-stage', stage);
  if (personName) {
    await page.selectOption('#new-task-people', { label: personName });
  }
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
  if (size) {
    await openTaskPage(page, title);
    await page.selectOption('#details-size', size);
    await page.click('.task-details-form button[type="submit"]');
    await ready(page);
    await page.locator('a', { hasText: 'Back to Board' }).click();
    await ready(page);
  }
}

function laneLocator(page: Page, name: string) {
  return page.locator('.team-lane', { has: page.locator('h2', { hasText: name }) });
}

// SPEC gates 2.24/2.25/2.27/2.30: lane order and contents, grouping,
// numbering, and workload blocks/text.
test('Team view: lane order, grouping, numbering, and workload blocks', async ({ page, server, browser }) => {
  const nameA = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await ready(page);

  const otherContext = await browser.newContext();
  const otherPage = await otherContext.newPage();
  const nameB = await signInAsNewPerson(otherPage, server.baseURL, '/tasks/board');
  await ready(otherPage);
  await otherContext.close();

  await page.reload();
  await ready(page);

  const unassignedTitle = uniqueName('Nobodys job');
  await addTask(page, unassignedTitle, 'todo');

  const doingTitle = uniqueName('Big current job');
  await addTask(page, doingTitle, 'doing', nameA, 'L');

  const upNext1 = uniqueName('First priority');
  await addTask(page, upNext1, 'todo', nameA, 'S');

  const upNext2 = uniqueName('Second priority');
  await addTask(page, upNext2, 'todo', nameA, 'S');

  const sharedTitle = uniqueName('Shared job');
  await openNewTask(page);
  await page.fill('#new-task-title', sharedTitle);
  await page.selectOption('#new-task-stage', 'todo');
  await page.selectOption('#new-task-people', [{ label: nameA }, { label: nameB }]);
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);

  await page.goto(server.baseURL + '/tasks/team');
  await ready(page);

  // Lane order: Unassigned first, then alphabetical.
  const laneNames = await page.locator('.team-lane h2').allTextContents();
  const relevant = laneNames.filter((n) => n === 'Unassigned' || n === nameA || n === nameB);
  const expectedOrder = ['Unassigned', nameA, nameB].sort((a, b) => {
    if (a === 'Unassigned') return -1;
    if (b === 'Unassigned') return 1;
    return a.localeCompare(b);
  });
  expect(relevant).toEqual(expectedOrder);

  // Unassigned has no workload row, only Up next.
  const unassignedLane = laneLocator(page, 'Unassigned');
  await expect(unassignedLane.locator('.workload')).toHaveCount(0);
  await expect(unassignedLane.locator('.team-task-list li', { hasText: unassignedTitle })).toHaveCount(1);

  // A's lane: Working on now has the doing task; Up next is numbered.
  const laneA = laneLocator(page, nameA);
  await expect(laneA.locator('h3', { hasText: 'Working on now' })).toHaveCount(1);
  await expect(laneA.locator('.team-task-list', { hasText: doingTitle })).toHaveCount(1);

  const upNextItems = (
    await laneA.locator('h3', { hasText: 'Up next' }).locator('xpath=following-sibling::ul[1]/li').allTextContents()
  ).map((t) => t.trim());
  // The priority number is a large figure before the title (gate 4.16), not "1." text.
  expect(upNextItems.some((t) => t.startsWith('1') && t.includes(upNext1))).toBe(true);
  expect(upNextItems.some((t) => t.startsWith('2') && t.includes(upNext2))).toBe(true);
  expect(upNextItems.some((t) => t.startsWith('3') && t.includes(sharedTitle))).toBe(true);

  // Workload: one L (4) + one M, default size, from the shared job (2)
  // + two S (1 each) = 8 blocks, "1 large · 1 medium · 2 small".
  await expect(laneA.locator('.workload-line')).toHaveText('1 large · 1 medium · 2 small');
  const blockCount = await laneA.locator('.workload-block').count();
  expect(blockCount).toBe(8);

  // Shared job also appears in B's lane (gate 2.30).
  const laneB = laneLocator(page, nameB);
  await expect(laneB.locator('.team-task-list li', { hasText: sharedTitle })).toHaveCount(1);
  await expect(laneB.locator('.team-task-list li', { hasText: upNext1 })).toHaveCount(0);
});

test('standard page checks for the Team view', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/team');
  await ready(page);
  await axeCheck(page);
  await expectNoSideScroll(page);
});
