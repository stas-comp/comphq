import { expect, test } from '../helpers/fixtures';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

// SPEC gates 8.30-8.34 (B14.3, D-106): steps, articles and places in the Add
// task window, saved with the job in one go.

test.describe.configure({ timeout: 90_000 });

const win = (page: Page) => page.locator('#task-window');
const FOLDER = '\\\\NAS\\COSMOS\\Mailings\\2026';

async function openAdd(page: Page, baseURL: string) {
  await page.goto(baseURL + '/tasks/board?person=mine');
  await ready(page);
  await page.click('#add-task-link');
  await expect(page.locator('#task-window[open]')).toBeVisible();
  await expect(win(page).locator('.new-steps .new-js')).toBeVisible();
}

async function addStep(page: Page, text: string, root: import('@playwright/test').Locator | Page = win(page)) {
  await root.locator('#new-step-text').fill(text);
  await root.locator('#new-step-text').press('Enter');
}

const stepWords = (page: Page) => win(page).locator('[data-new-step] .step-words').allTextContents();

async function newArticle(page: Page, baseURL: string, cat: string, title: string): Promise<string> {
  const res = await page.request.post(baseURL + '/kb/articles', {
    form: { title, category_id: cat, body_html: `<p>${title}</p>` },
    headers: { origin: baseURL },
  });
  expect(res.ok()).toBe(true);
  return /\/kb\/articles\/(\d+)/.exec(res.url())![1];
}

async function newCategory(page: Page, baseURL: string, name: string): Promise<string> {
  await page.request.post(baseURL + '/kb/categories', { form: { name }, headers: { origin: baseURL } });
  await page.goto(baseURL + '/kb/new');
  return (await page.locator('select[name="category_id"] option', { hasText: name }).getAttribute('value'))!;
}

test('gates 8.30, 8.31: build a job with five steps (reworded, reordered, one removed), two articles and a place, then save it all at once', async ({ page, server }) => {
  const me = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  const tag = uniqueName('add').replace(/[^a-z0-9]/gi, '').toLowerCase();
  const cat = await newCategory(page, server.baseURL, `Cat ${tag}`);
  const sop = `Mailing SOP ${tag}`;
  const lists = `Mailing lists ${tag}`;
  await newArticle(page, server.baseURL, cat, sop);
  await newArticle(page, server.baseURL, cat, lists);

  await openAdd(page, server.baseURL);
  const title = `Quarterly mailing ${tag}`;
  await win(page).locator('#new-task-title').fill(title);
  await win(page).locator('#new-task-people').selectOption({ label: me }); // so the board's "My tasks" filter shows it

  // Steps: each Enter adds one, the box clears and stays ready.
  for (const s of ['Pick the list', 'Print labels', 'Stuff envelopes', 'Post them', 'Count returns', 'Throw this one away']) await addStep(page, s);
  await expect(win(page).locator('#new-step-text')).toHaveValue('');
  await expect(win(page).locator('#new-step-text')).toBeFocused();
  expect(await stepWords(page)).toEqual(['Pick the list', 'Print labels', 'Stuff envelopes', 'Post them', 'Count returns', 'Throw this one away']);
  // Nothing has been saved: the job doesn't exist yet.
  const list = win(page).locator('[data-new-steps]');

  // Remove one, reword one, move one with the arrows.
  await win(page).getByRole('button', { name: 'Remove step “Throw this one away”' }).click();
  await win(page).locator('[data-new-step]', { hasText: 'Count returns' }).getByRole('button', { name: /Rename/ }).click();
  await win(page).locator('.step-rename input').fill('Count the returns');
  await win(page).locator('.step-rename input').press('Enter');
  await win(page).getByRole('button', { name: 'Move step “Pick the list” down' }).click();
  expect(await stepWords(page)).toEqual(['Print labels', 'Pick the list', 'Stuff envelopes', 'Post them', 'Count the returns']);
  await expect(list.locator('.step')).toHaveCount(5);

  // Two articles: type, pick.
  const search = win(page).locator('#new-article-q');
  await search.fill(`mailing ${tag}`);
  await win(page).locator('.ref-match-pick', { hasText: sop }).click();
  await search.fill(`mailing ${tag}`);
  await expect(win(page).locator('.ref-match-pick')).toHaveCount(1); // the one already chosen isn't offered again
  await win(page).locator('.ref-match-pick', { hasText: lists }).click();
  await expect(win(page).locator('[data-new-article]')).toHaveCount(2);

  // A place.
  await win(page).locator('#new-place-path').fill(' //NAS/COSMOS/Mailings/2026 ');
  await win(page).locator('#new-place-name').fill('Mailing lists 2026');
  await win(page).getByRole('button', { name: 'Add place' }).click();
  await expect(win(page).locator('[data-new-place] .ref-path')).toHaveText(FOLDER);

  await win(page).getByRole('button', { name: 'Add task' }).click();
  await expect(page.locator('#task-window[open]')).toHaveCount(0);
  const card = page.locator('.task-card', { hasText: title });
  await expect(card).toBeVisible();
  await expect(card.locator('.step-badge')).toContainText('0/5');

  // Opening the new job shows them all, in the order shown.
  await card.locator('.task-card-title a').click();
  await expect(win(page).locator('.task-steps .step-words')).toHaveText(['Print labels', 'Pick the list', 'Stuff envelopes', 'Post them', 'Count the returns']);
  await expect(win(page).locator('.ref-articles .ref-line')).toHaveCount(2);
  await expect(win(page).locator('.ref-articles')).toContainText(sop);
  await expect(win(page).locator('.ref-articles')).toContainText(lists);
  await expect(win(page).locator('.ref-places .ref-place', { hasText: 'Mailing lists 2026' }).locator('.ref-path')).toHaveText(FOLDER);

  // One History line for the new job.
  await win(page).locator('.task-history summary').click();
  const history = await win(page).locator('.task-activity-list li').allTextContents();
  expect(history).toHaveLength(1);
  expect(history[0]).toContain('created this (with 5 steps)');
});

test('gate 8.30: steps can be dragged into order before saving', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await openAdd(page, server.baseURL);
  for (const s of ['First', 'Second', 'Third']) await addStep(page, s);
  const third = win(page).locator('[data-new-step]', { hasText: 'Third' }).locator('.step-body');
  const first = win(page).locator('[data-new-step]', { hasText: 'First' }).locator('.step-body');
  const tb = (await third.boundingBox())!;
  const fb = (await first.boundingBox())!;
  await page.mouse.move(tb.x + tb.width / 2, tb.y + tb.height / 2);
  await page.mouse.down();
  await page.mouse.move(tb.x + tb.width / 2, tb.y + tb.height / 2 - 6, { steps: 4 });
  for (let i = 1; i <= 10; i++) {
    await page.mouse.move(fb.x + fb.width / 2, tb.y + ((fb.y + 4 - tb.y) * i) / 10, { steps: 2 });
    await page.waitForTimeout(60);
  }
  await page.mouse.up();
  await expect.poll(() => stepWords(page)).toEqual(['Third', 'First', 'Second']);
});

test('gate 8.32: every refusal saves nothing and keeps what was typed', async ({ page, server }) => {
  const me = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  void me;
  const tag = uniqueName('ref').replace(/[^a-z0-9]/gi, '').toLowerCase();
  await openAdd(page, server.baseURL);
  const title = `Refused ${tag}`;
  await win(page).locator('#new-task-title').fill(title);
  await addStep(page, 'Keep this step');
  await addStep(page, 'And this one');
  await win(page).locator('#new-place-path').fill(FOLDER);
  await win(page).getByRole('button', { name: 'Add place' }).click();

  // A bad folder path that gets past the form (here, a hidden field edited by hand).
  await win(page).locator('.add-task-form').evaluate((form) => {
    const bad = document.createElement('input');
    bad.type = 'hidden';
    bad.name = 'place_path';
    bad.value = 'nonsense';
    form.appendChild(bad);
    const name = document.createElement('input');
    name.type = 'hidden';
    name.name = 'place_name';
    name.value = '';
    form.appendChild(name);
  });
  await win(page).getByRole('button', { name: 'Add task' }).click();
  await expect(win(page).locator('[role="alert"]').first()).toContainText("That doesn't look like a folder path.");
  // Everything is still there: the title, both steps and the good place (and the bad one as typed).
  await expect(win(page).locator('#new-task-title')).toHaveValue(title);
  expect(await stepWords(page)).toEqual(['Keep this step', 'And this one']);
  await expect(win(page).locator('[data-new-place] .ref-path').first()).toHaveText(FOLDER);

  // No title, with steps typed: refused, and the steps stay.
  await win(page).locator('#new-task-title').fill('   ');
  await win(page).locator('[data-new-place]').nth(1).getByRole('button', { name: /Remove place/ }).click();
  await win(page).getByRole('button', { name: 'Add task' }).click();
  await expect(win(page).locator('[role="alert"]').first()).toContainText('Please type a title.');
  expect(await stepWords(page)).toEqual(['Keep this step', 'And this one']);

  // Nothing was saved by any of it.
  await page.reload();
  await ready(page);
  await expect(page.locator('.task-card', { hasText: title })).toHaveCount(0);
});

test('gate 8.32: a step over 200 characters, more than 50 steps and more than 20 places are refused in plain words', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await openAdd(page, server.baseURL);
  const long = 'x'.repeat(201);
  await win(page).locator('#new-step-text').fill(long);
  await win(page).getByRole('button', { name: 'Add', exact: true }).click();
  await expect(win(page).locator('.new-steps .new-notice')).toHaveText('A step can be up to 200 characters.');
  await expect(win(page).locator('[data-new-step]')).toHaveCount(0);

  for (let i = 1; i <= 50; i++) await addStep(page, `Step ${i}`);
  await addStep(page, 'One too many');
  await expect(win(page).locator('.new-steps .new-notice')).toHaveText('A job can have up to 50 steps.');
  await expect(win(page).locator('[data-new-step]')).toHaveCount(50);

  // The server says the same when the form is edited past its own limits.
  const postForm = (pairs: [string, string][]) =>
    page.request.post(server.baseURL + '/tasks', {
      data: new URLSearchParams(pairs).toString(),
      headers: { origin: server.baseURL, 'content-type': 'application/x-www-form-urlencoded' },
    });
  const res = await postForm([['title', 'Too many steps'], ['fragment', '1'], ...Array.from({ length: 51 }, (_, i): [string, string] => ['step', `S${i}`])]);
  expect(res.status()).toBe(422);
  expect(await res.text()).toContain('A job can have up to 50 steps.');
  const long2 = await postForm([['title', 'Long step'], ['fragment', '1'], ['step', long]]);
  expect(long2.status()).toBe(422);
  expect(await long2.text()).toContain('A step can be up to 200 characters.');
  const placePairs: [string, string][] = [];
  for (let i = 0; i < 21; i++) placePairs.push(['place_name', ''], ['place_path', `\\\\NAS\\COSMOS\\F${i}`]);
  const places = await postForm([['title', 'Many places'], ['fragment', '1'], ...placePairs]);
  expect(places.status()).toBe(422);
  expect(await places.text()).toContain('A job can have at most 20 COSMOS places.');
});

test('gate 8.33: Cancel and closing save nothing, and ask "Discard this new job?" only when steps, articles or places were added', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  const tag = uniqueName('disc').replace(/[^a-z0-9]/gi, '').toLowerCase();
  const messages: string[] = [];
  let accept = false;
  page.on('dialog', async (dialog) => {
    messages.push(dialog.message());
    if (accept) await dialog.accept();
    else await dialog.dismiss();
  });

  // With a step in the form: Cancel asks, and "keep editing" leaves the window as it was.
  await openAdd(page, server.baseURL);
  await win(page).locator('#new-task-title').fill(`Discard ${tag}`);
  await addStep(page, 'A step');
  await win(page).getByRole('button', { name: 'Cancel' }).click();
  expect(messages).toEqual(['Discard this new job?']);
  await expect(page.locator('#task-window[open]')).toBeVisible();
  expect(await stepWords(page)).toEqual(['A step']);
  // The close icon asks the same, and so does Escape.
  await win(page).getByRole('button', { name: 'Close' }).click();
  await page.keyboard.press('Escape');
  expect(messages).toEqual(['Discard this new job?', 'Discard this new job?', 'Discard this new job?']);
  // Yes: it closes, and nothing was saved.
  accept = true;
  await win(page).getByRole('button', { name: 'Cancel' }).click();
  await expect(page.locator('#task-window[open]')).toHaveCount(0);
  await page.reload();
  await ready(page);
  await expect(page.locator('.task-card', { hasText: `Discard ${tag}` })).toHaveCount(0);

  // With only a title typed it is the old question, and with nothing at all it just closes.
  messages.length = 0;
  accept = true;
  await openAdd(page, server.baseURL);
  await win(page).locator('#new-task-title').fill('Only a title');
  await win(page).getByRole('button', { name: 'Cancel' }).click();
  expect(messages).toEqual(['Leave without saving?']);
  await openAdd(page, server.baseURL);
  await win(page).getByRole('button', { name: 'Cancel' }).click();
  expect(messages).toEqual(['Leave without saving?']);
  await expect(page.locator('#task-window[open]')).toHaveCount(0);

  // A place or an article alone counts too.
  messages.length = 0;
  await openAdd(page, server.baseURL);
  await win(page).locator('#new-place-path').fill(FOLDER);
  await win(page).getByRole('button', { name: 'Add place' }).click();
  await win(page).getByRole('button', { name: 'Cancel' }).click();
  expect(messages).toEqual(['Discard this new job?']);
});

test('gate 8.34: without JavaScript the Add task page has a Steps box (one per line, blanks ignored), and says articles and places come after saving', async ({ browser, server }) => {
  const context = await browser.newContext({ javaScriptEnabled: false });
  const page = await context.newPage();
  const normal = await browser.newContext();
  const p = await normal.newPage();
  await signInAsNewPerson(p, server.baseURL, '/tasks/board');
  await context.addCookies(await normal.cookies());
  await normal.close();

  const title = `No script ${uniqueName('n').replace(/[^a-z0-9]/gi, '')}`;
  await page.goto(server.baseURL + '/tasks/new');
  await expect(page.getByLabel('Steps, one on each line')).toBeVisible();
  await expect(page.getByText("Articles and COSMOS places can be added on the job's page after you save it.")).toBeVisible();
  await expect(page.locator('#new-step-text')).toBeHidden();
  await page.getByLabel('Title').fill(title);
  await page.getByLabel('Steps, one on each line').fill('First thing\n\n   \nSecond thing\r\nThird thing\n');
  await page.getByRole('button', { name: 'Add task' }).click();

  await page.goto(server.baseURL + '/tasks/board?q=' + encodeURIComponent(title));
  const href = (await page.locator('.task-card', { hasText: title }).locator('.task-card-title a').getAttribute('href'))!;
  await page.goto(server.baseURL + href);
  await expect(page.locator('.steps-list .step-words')).toHaveText(['First thing', 'Second thing', 'Third thing']);
  const history = await page.locator('.task-activity-list li').allTextContents();
  expect(history.join(' ')).toContain('created this (with 3 steps)');

  // A refusal keeps the box as typed.
  await page.goto(server.baseURL + '/tasks/new');
  await page.getByLabel('Title').fill('   ');
  await page.getByLabel('Steps, one on each line').fill('Kept one\nKept two');
  await page.getByRole('button', { name: 'Add task' }).click();
  await expect(page.locator('.message[role="alert"]').first()).toContainText('Please type a title.');
  await expect(page.getByLabel('Steps, one on each line')).toHaveValue('Kept one\nKept two');
  await context.close();
});

test('standard checks: axe and no sideways scroll with steps, an article and a place in the window and on the page', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  await openAdd(page, server.baseURL);
  await addStep(page, 'A step');
  await win(page).locator('#new-place-path').fill(FOLDER);
  await win(page).getByRole('button', { name: 'Add place' }).click();
  await axeCheck(page);
  await expectNoSideScroll(page);
  await page.goto(server.baseURL + '/tasks/new');
  await ready(page);
  await expect(page.locator('.new-steps .new-js')).toBeVisible();
  await addStep(page, 'A step', page);
  await axeCheck(page);
  await expectNoSideScroll(page);
});

test('the page version works too: steps and a place added with script, then saved', async ({ page, server }) => {
  const me = await signInAsNewPerson(page, server.baseURL, '/tasks/board');
  void me;
  const title = `Page ${uniqueName('p').replace(/[^a-z0-9]/gi, '')}`;
  await page.goto(server.baseURL + '/tasks/new');
  await ready(page);
  await page.fill('#new-task-title', title);
  await addStep(page, 'One', page);
  await addStep(page, 'Two', page);
  await page.fill('#new-place-path', 'S:/COSMOS/Proofs');
  await page.getByRole('button', { name: 'Add place' }).click();
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
  await page.goto(server.baseURL + '/tasks/board?q=' + encodeURIComponent(title));
  const href = (await page.locator('.task-card', { hasText: title }).locator('.task-card-title a').getAttribute('href'))!;
  await page.goto(server.baseURL + href);
  await expect(page.locator('.steps-list .step-words')).toHaveText(['One', 'Two']);
  await expect(page.locator('.ref-places .ref-path')).toHaveText('S:\\COSMOS\\Proofs');
});
