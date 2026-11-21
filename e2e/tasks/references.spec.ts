import { expect, test } from '../helpers/fixtures';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { idFromHref, myPersonID, seedJob } from '../helpers/tasks';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

// SPEC gates 8.10-8.14, 8.18, 8.19 (B14.2, D-105): articles and COSMOS places
// on a job, on screen.

test.describe.configure({ timeout: 60_000 });

const section = (page: Page) => page.locator('.task-refs');
const card = (page: Page, title: string) => page.locator('.task-card', { hasText: title });

async function setup(page: Page, baseURL: string) {
  await signInAsNewPerson(page, baseURL, '/tasks/board');
  const me = await myPersonID(page, baseURL);
  const tag = uniqueName('ref').replace(/[^a-z0-9]/gi, '').toLowerCase();
  const job = `Send the mailing ${tag}`;
  await seedJob(page, baseURL, { title: job, stage: 'todo', personID: me });
  await page.goto(baseURL + '/tasks/board?person=mine');
  await ready(page);
  const id = idFromHref((await card(page, job).locator('.task-card-title a').first().getAttribute('href'))!);
  return { tag, job, id };
}

async function newCategory(page: Page, baseURL: string, name: string): Promise<string> {
  const res = await page.request.post(baseURL + '/kb/categories', { form: { name }, headers: { origin: baseURL } });
  expect(res.ok()).toBe(true);
  await page.goto(baseURL + '/kb/new');
  return (await page.locator('select[name="category_id"] option', { hasText: name }).getAttribute('value'))!;
}

async function publish(page: Page, baseURL: string, categoryID: string, title: string): Promise<string> {
  const res = await page.request.post(baseURL + '/kb/articles', {
    form: { title, category_id: categoryID, body_html: `<p>${title}</p>` },
    headers: { origin: baseURL },
  });
  expect(res.ok()).toBe(true);
  return /\/kb\/articles\/(\d+)/.exec(res.url())![1];
}

async function openWindow(page: Page, baseURL: string, title: string) {
  await page.goto(baseURL + '/tasks/board?person=mine');
  await ready(page);
  await card(page, title).locator('.task-card-title a').click();
  await expect(page.locator('#task-window[open]')).toBeVisible();
  await expect(section(page)).toBeVisible();
}

const FOLDER = '\\\\NAS\\COSMOS\\Mailings\\2026';

test('gate 8.10: the window and the page have Articles and places under Linked jobs, with both controls', async ({ page, server }) => {
  const { job, id } = await setup(page, server.baseURL);
  await openWindow(page, server.baseURL, job);
  const headings = await page.locator('#task-window h2').allTextContents();
  expect(headings.indexOf('Articles and places')).toBeGreaterThan(headings.indexOf('Linked jobs'));
  await expect(section(page).getByLabel('Link an article')).toBeVisible();
  await expect(section(page).getByText('Add a COSMOS place')).toBeVisible();

  await page.goto(`${server.baseURL}/tasks/${id}`);
  await ready(page);
  const pageHeadings = await page.locator('h2').allTextContents();
  expect(pageHeadings.indexOf('Articles and places')).toBeGreaterThan(pageHeadings.indexOf('Linked jobs'));
  await expect(section(page).getByLabel('Link an article')).toBeVisible();
  await expect(section(page).getByLabel('Folder path')).toBeVisible();
});

test('gates 8.11, 8.12: the picker lists matching published articles, linking is at once, archived are not offered', async ({ page, server }) => {
  const { job, tag, id } = await setup(page, server.baseURL);
  const catName = `Cat ${tag}`;
  const cat = await newCategory(page, server.baseURL, catName);
  const sop = `Mailing SOP ${tag}`;
  const lists = `Mailing lists ${tag}`;
  const old = `Mailing old ${tag}`;
  await publish(page, server.baseURL, cat, sop);
  await publish(page, server.baseURL, cat, lists);
  const oldID = await publish(page, server.baseURL, cat, old);
  await page.request.post(`${server.baseURL}/kb/articles/${oldID}/archive`, { headers: { origin: server.baseURL } });

  await openWindow(page, server.baseURL, job);
  await section(page).getByLabel('Link an article').fill(`mailing ${tag}`);
  const picks = section(page).locator('.ref-match-pick');
  await expect(picks).toHaveCount(2); // the archived one is not offered
  await expect(section(page).locator('.ref-results')).toContainText(catName); // each with its category
  await expect(section(page).locator('.ref-results')).not.toContainText(old);

  // Picking one links it straight away, and then it is no longer offered.
  await picks.filter({ hasText: sop }).click();
  const line = section(page).locator('.ref-articles .ref-line', { hasText: sop });
  await expect(line).toBeVisible();
  await expect(line.getByRole('link', { name: sop })).toHaveAttribute('href', /\/kb\/articles\/\d+/);
  await expect(line).toContainText(catName);
  await section(page).getByLabel('Link an article').fill(`mailing ${tag}`);
  await expect(section(page).locator('.ref-match-pick')).toHaveCount(1);
  await expect(section(page).locator('.ref-results')).not.toContainText(sop);

  // The same article twice is refused in plain words, from a stale picker.
  const dup = await page.request.post(`${server.baseURL}/tasks/${id}/articles`, {
    form: { article_id: (await line.getByRole('link').getAttribute('href'))!.split('/').pop()!, fragment: '1' },
    headers: { origin: server.baseURL },
  });
  expect(await dup.text()).toContain('That article is already linked to this job.');
});

test('gate 8.12: an article archived after linking stays on the job, marked Archived, and still opens', async ({ page, server }) => {
  const { job, tag, id } = await setup(page, server.baseURL);
  const cat = await newCategory(page, server.baseURL, `Cat ${tag}`);
  const title = `Mailing SOP ${tag}`;
  const articleID = await publish(page, server.baseURL, cat, title);
  const linked = await page.request.post(`${server.baseURL}/tasks/${id}/articles`, { form: { article_id: articleID }, headers: { origin: server.baseURL } });
  expect(linked.ok()).toBe(true);

  await page.request.post(`${server.baseURL}/kb/articles/${articleID}/archive`, { headers: { origin: server.baseURL } });
  await openWindow(page, server.baseURL, job);
  const line = section(page).locator('.ref-articles .ref-line', { hasText: title });
  await expect(line.locator('.ref-line-archived')).toHaveText('Archived');
  await line.getByRole('link', { name: title }).click();
  await ready(page);
  await expect(page.locator('h1')).toHaveText(title);
});

test('gates 8.13, 8.14: add a place with and without a name; each shows name, path and Copy, which copies the exact path', async ({ page, server, context }) => {
  await context.grantPermissions(['clipboard-read', 'clipboard-write']);
  const { job } = await setup(page, server.baseURL);
  await openWindow(page, server.baseURL, job);

  await section(page).getByLabel('Folder path').fill(` //NAS/COSMOS/Mailings/2026 `);
  await section(page).getByLabel('Name (optional)').fill('Mailing lists 2026');
  await section(page).getByRole('button', { name: 'Add place' }).click();
  const named = section(page).locator('.ref-places .ref-place', { hasText: 'Mailing lists 2026' });
  await expect(named).toBeVisible();
  await expect(named.locator('.ref-path')).toHaveText(FOLDER); // normalised: slashes turned, ends trimmed
  const mono = await named.locator('.ref-path').evaluate((el) => getComputedStyle(el).fontFamily);
  expect(mono.toLowerCase()).toMatch(/mono|consolas|courier/);

  // With no name, the last folder is the name.
  await section(page).getByLabel('Folder path').fill('S:\\COSMOS\\Proofs');
  await section(page).getByRole('button', { name: 'Add place' }).click();
  const unnamed = section(page).locator('.ref-places .ref-place', { hasText: 'S:\\COSMOS\\Proofs' });
  await expect(unnamed.locator('.ref-line-title')).toHaveText('Proofs');

  // Copy puts the exact path on the clipboard, and reads Copied for 2 seconds.
  await named.getByRole('button', { name: 'Copy' }).click();
  await expect(named.locator('.copy-button')).toHaveText('Copied');
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(FOLDER);
  await expect(named.locator('.copy-button')).toHaveText('Copy', { timeout: 5000 });
  await expect(section(page).locator('.ref-hint')).toHaveText("Paste a path into File Explorer's address bar to open the folder.");
});

test('gate 8.14: Copy works on a page with no clipboard API (the NAS\'s ordinary address), through the browser\'s copy command', async ({ page, server }) => {
  const { job } = await setup(page, server.baseURL);
  // As on an http:// page with no padlock: no navigator.clipboard at all. The
  // browser's own copy command is recorded, with the text that was selected.
  await page.addInitScript(() => {
    Object.defineProperty(navigator, 'clipboard', { value: undefined, configurable: true });
    (window as unknown as { __copied: string[] }).__copied = [];
    document.execCommand = (cmd: string) => {
      const box = document.activeElement as HTMLTextAreaElement | null;
      if (cmd === 'copy' && box && 'value' in box) {
        (window as unknown as { __copied: string[] }).__copied.push(box.value);
        return true;
      }
      return false;
    };
  });
  await openWindow(page, server.baseURL, job);
  await section(page).getByLabel('Folder path').fill(FOLDER);
  await section(page).getByRole('button', { name: 'Add place' }).click();
  const place = section(page).locator('.ref-place');
  await place.getByRole('button', { name: 'Copy' }).click();
  await expect(place.locator('.copy-button')).toHaveText('Copied');
  expect(await page.evaluate(() => (window as unknown as { __copied: string[] }).__copied)).toEqual([FOLDER]);
});

test('gates 8.15, 8.16: refusals show plain words and keep what was typed', async ({ page, server }) => {
  const { job } = await setup(page, server.baseURL);
  await openWindow(page, server.baseURL, job);
  await section(page).getByLabel('Folder path').fill('COSMOS/Mailings');
  await section(page).getByRole('button', { name: 'Add place' }).click();
  await expect(section(page).locator('.refs-notice')).toHaveText(
    "That doesn't look like a folder path. It should start with \\\\ (like \\\\NAS\\COSMOS) or a drive letter (like S:\\COSMOS).",
  );
  await expect(section(page).getByLabel('Folder path')).toHaveValue('COSMOS/Mailings');
  await expect(section(page).locator('.ref-place')).toHaveCount(0);

  await section(page).getByLabel('Folder path').fill(FOLDER);
  await section(page).getByRole('button', { name: 'Add place' }).click();
  await expect(section(page).locator('.ref-place')).toHaveCount(1);
  await section(page).getByLabel('Folder path').fill(FOLDER.toLowerCase());
  await section(page).getByLabel('Name (optional)').fill('Again');
  await section(page).getByRole('button', { name: 'Add place' }).click();
  await expect(section(page).locator('.refs-notice')).toHaveText('That folder is already on this job.');
  await expect(section(page).getByLabel('Name (optional)')).toHaveValue('Again');
  await expect(section(page).locator('.ref-place')).toHaveCount(1);
});

test('gate 8.17: the remove icon takes an article or a place off the job, and History records each change', async ({ page, server }) => {
  const { job, tag, id } = await setup(page, server.baseURL);
  const cat = await newCategory(page, server.baseURL, `Cat ${tag}`);
  const title = `Mailing SOP ${tag}`;
  const articleID = await publish(page, server.baseURL, cat, title);
  await page.request.post(`${server.baseURL}/tasks/${id}/articles`, { form: { article_id: articleID }, headers: { origin: server.baseURL } });
  await page.request.post(`${server.baseURL}/tasks/${id}/places`, { form: { name: 'Mailing lists 2026', path: FOLDER }, headers: { origin: server.baseURL } });

  await openWindow(page, server.baseURL, job);
  await section(page).getByRole('button', { name: `Remove article ${title}` }).click();
  await expect(section(page).locator('.ref-articles .ref-line')).toHaveCount(0);
  await section(page).getByRole('button', { name: 'Remove place Mailing lists 2026' }).click();
  await expect(section(page).locator('.ref-place')).toHaveCount(0);

  await page.goto(`${server.baseURL}/tasks/${id}`);
  await ready(page);
  const history = await page.locator('.task-activity-list li').allInnerTexts();
  expect(history.some((t) => t.includes(`linked the article “${title}”`))).toBe(true);
  expect(history.some((t) => t.includes(`removed the article “${title}”`))).toBe(true);
  expect(history.some((t) => t.includes('added the COSMOS place “Mailing lists 2026”'))).toBe(true);
  expect(history.some((t) => t.includes('removed the COSMOS place “Mailing lists 2026”'))).toBe(true);
});

test('gate 8.19: without JavaScript the article picker, linking, unlinking and places are all ordinary forms', async ({ browser, server }) => {
  const context = await browser.newContext({ javaScriptEnabled: false });
  const page = await context.newPage();
  const { tag, id } = await (async () => {
    // Set up with a normal page, then reuse the same person in the no-script context.
    const normal = await browser.newContext();
    const p = await normal.newPage();
    const made = await setup(p, server.baseURL);
    const cookies = await normal.cookies();
    await context.addCookies(cookies);
    const cat = await newCategory(p, server.baseURL, `Cat ${made.tag}`);
    await publish(p, server.baseURL, cat, `Mailing SOP ${made.tag}`);
    await normal.close();
    return made;
  })();
  const title = `Mailing SOP ${tag}`;

  await page.goto(`${server.baseURL}/tasks/${id}`);
  await page.getByLabel('Link an article').fill(`mailing ${tag}`);
  await page.locator('.ref-search').getByRole('button', { name: 'Find' }).click();
  await expect(page.locator('.ref-match', { hasText: title })).toBeVisible();
  await page.locator('.ref-match', { hasText: title }).getByRole('button', { name: 'Link' }).click();
  await expect(page.locator('.ref-articles .ref-line', { hasText: title })).toBeVisible();

  await page.getByLabel('Folder path').fill('S:/COSMOS/Proofs');
  await page.getByRole('button', { name: 'Add place' }).click();
  const place = page.locator('.ref-place');
  await expect(place.locator('.ref-path')).toHaveText('S:\\COSMOS\\Proofs'); // plain text, selectable by hand

  // A refusal comes back on the page with the typed words kept.
  await page.getByLabel('Folder path').fill('nonsense');
  await page.getByRole('button', { name: 'Add place' }).click();
  await expect(page.locator('.message[role="alert"]').first()).toContainText("That doesn't look like a folder path.");
  await expect(page.getByLabel('Folder path')).toHaveValue('nonsense');

  await page.getByRole('button', { name: `Remove article ${title}` }).click();
  await expect(page.locator('.ref-articles .ref-line')).toHaveCount(0);
  await page.getByRole('button', { name: 'Remove place Proofs' }).click();
  await expect(page.locator('.ref-place')).toHaveCount(0);
  await context.close();
});

test('standard checks: axe with the section open, in the window and on the page, and no sideways scroll at both window sizes', async ({ page, server }) => {
  const { job, id } = await setup(page, server.baseURL);
  await page.request.post(`${server.baseURL}/tasks/${id}/places`, { form: { name: 'Lists', path: FOLDER }, headers: { origin: server.baseURL } });
  await openWindow(page, server.baseURL, job);
  await section(page).getByLabel('Link an article').fill('zz');
  await axeCheck(page);
  await expectNoSideScroll(page);
  await page.goto(`${server.baseURL}/tasks/${id}`);
  await ready(page);
  await axeCheck(page);
  await expectNoSideScroll(page);
});
