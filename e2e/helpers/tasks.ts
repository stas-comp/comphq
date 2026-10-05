import type { Page } from '@playwright/test';
import { ready } from './ready';

// The board no longer carries an add-a-task form (SPEC gate 4.15): + Add
// task opens the task window, and the same form is always available as a
// page (gate 4.28). Tests that create a task open that page, fill the
// fields (ids new-task-title, -notes, -size, -stage, -due-date, -people),
// and press the form's submit button (.add-task-form button[type=submit]),
// which lands back on the Board.
export async function openNewTask(page: Page): Promise<void> {
  await page.goto(new URL('/tasks/new', page.url()).href);
  await ready(page);
}

// Clicking a card's title opens the task window (SPEC gate 4.23); the same
// link, followed as a link, is the task's own page (gate 4.27). Tests that
// mean the page go there by the link's address.
export async function openTaskPage(page: Page, title: string): Promise<void> {
  const href = await page.locator('.task-card', { hasText: title }).locator('.task-card-title a').first().getAttribute('href');
  if (!href) throw new Error(`no card titled ${title}`);
  await page.goto(new URL(href, page.url()).href);
  await ready(page);
}

/** The signed-in person's id, from their cookie. */
export async function myPersonID(page: Page, baseURL: string): Promise<string> {
  const cookie = (await page.context().cookies(baseURL)).find((c) => c.name === 'comphq_person');
  if (!cookie) throw new Error('not signed in');
  return cookie.value;
}

/**
 * Makes jobs straight through the form's own endpoint (much quicker than
 * the window for a long column). They are all on `personID` when given.
 */
export async function seedJobs(
  page: Page,
  baseURL: string,
  titles: string[],
  stage: 'idea' | 'todo' | 'doing' | 'done',
  personID?: string,
): Promise<void> {
  for (const title of titles) {
    const form: Record<string, string> = { title, stage };
    if (personID) form.person_id = personID;
    const res = await page.request.post(baseURL + '/tasks', { form, headers: { origin: baseURL }, maxRedirects: 0 });
    if (res.status() >= 400) throw new Error(`could not make job ${title}: ${res.status()}`);
  }
}
