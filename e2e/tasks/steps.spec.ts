import { expect, test } from '../helpers/fixtures';
import { axeCheck } from '../helpers/axe';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { openNewTask, openTaskPage } from '../helpers/tasks';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

// Steps inside a job, with script (PLAN P5-03, gates 5.01-5.07, 5.09, 5.12,
// 5.16). Every gate is proved twice: once in the task window over the Board
// and once on the task's own page, because they draw one partial (SPEC B10.3)
// and must never disagree. The plain-form layer underneath is tested with
// script switched off in steps-no-script.spec.ts.

type Home = 'window' | 'page';
const homes: Home[] = ['window', 'page'];

const steps = (page: Page) => page.locator('.task-steps');
const words = (page: Page) => page.locator('.task-steps .step-words');
const box = (page: Page) => page.getByPlaceholder('Add a step');
const rowOf = (page: Page, text: string) =>
  page.locator('.task-steps .step', { has: page.locator('.step-words', { hasText: text }) });
const checkOf = (page: Page, text: string) => steps(page).getByRole('checkbox', { name: text, exact: true });
const count = (page: Page) => steps(page).locator('.steps-count');

/** Makes a job as a new person and opens it in `home`. */
async function openJob(page: Page, baseURL: string, home: Home): Promise<{ name: string; title: string }> {
  const name = await signInAsNewPerson(page, baseURL, '/tasks/board');
  await ready(page);
  const title = uniqueName('Concert');
  await openNewTask(page);
  await page.fill('#new-task-title', title);
  await page.click('.add-task-form button[type="submit"]');
  await ready(page);
  await openIn(page, title, home);
  return { name, title };
}

async function openIn(page: Page, title: string, home: Home): Promise<void> {
  if (home === 'page') {
    await openTaskPage(page, title);
  } else {
    await page.locator('.task-card', { hasText: title }).locator('.task-card-title a').first().click();
    await expect(page.locator('#task-window')).toBeVisible();
  }
  await expect(steps(page)).toBeVisible();
  // A marker that survives only if the page is never reloaded.
  await page.evaluate(() => ((window as unknown as { __kept: boolean }).__kept = true));
}

const notReloaded = (page: Page) => page.evaluate(() => (window as unknown as { __kept?: boolean }).__kept === true);

/** Types steps straight through with Enter, never pausing, and waits for them all. */
async function typeSteps(page: Page, ...texts: string[]): Promise<void> {
  const before = await words(page).allTextContents();
  await box(page).click();
  for (const t of texts) {
    await page.keyboard.type(t);
    await page.keyboard.press('Enter');
  }
  await expect(words(page)).toHaveText([...before, ...texts]);
}

for (const home of homes) {
  test.describe(`in the ${home}`, () => {
    test(`gate 5.01: a job with no steps shows Steps and an Add a step line and nothing else (${home})`, async ({ page, server }) => {
      await openJob(page, server.baseURL, home);
      await expect(steps(page).getByRole('heading', { name: 'Steps' })).toBeVisible();
      await expect(box(page)).toBeVisible();
      await expect(steps(page).getByRole('button', { name: 'Add' })).toBeVisible();
      await expect(page.locator('.task-steps .steps-list, .task-steps .steps-count, .task-steps progress')).toHaveCount(0);
    });

    test(`gate 5.02: steps are typed one after another, Enter each time, and the box stays ready (${home})`, async ({ page, server }) => {
      await openJob(page, server.baseURL, home);
      await box(page).click();
      const typed = ['Print exam papers', 'Book the hall', 'Email the parents', 'Order programmes'];
      for (const t of typed) {
        await page.keyboard.type(t);
        await page.keyboard.press('Enter'); // straight on to the next, never pausing for the last one
      }
      await expect(words(page)).toHaveText(typed);
      await expect(box(page)).toBeFocused();
      await expect(box(page)).toHaveValue('');
      await expect(count(page)).toHaveText('0 of 4 done');
      expect(await notReloaded(page)).toBe(true);
    });

    test(`gate 5.02: pressing Add does the same as Enter (${home})`, async ({ page, server }) => {
      await openJob(page, server.baseURL, home);
      await box(page).fill('Book the hall');
      await steps(page).getByRole('button', { name: 'Add' }).click();
      await expect(words(page)).toHaveText(['Book the hall']);
      await expect(box(page)).toHaveValue('');
    });

    test(`gate 5.03, 5.04: ticking is at once, shows who and when, and the heading and bar follow (${home})`, async ({ page, server }) => {
      const { name } = await openJob(page, server.baseURL, home);
      await typeSteps(page, 'One', 'Two', 'Three');

      await checkOf(page, 'One').check();
      await expect(count(page)).toHaveText('1 of 3 done');
      await expect(checkOf(page, 'One')).toBeChecked();
      await expect(rowOf(page, 'One').locator('.step-by')).toHaveText(
        new RegExp(`^${name}, (Mon|Tue|Wed|Thu|Fri|Sat|Sun) \\d{1,2} [A-Z][a-z]{2}`),
      );
      await expect(rowOf(page, 'Two').locator('.step-by')).toHaveCount(0);
      const bar = steps(page).locator('progress');
      await expect(bar).toHaveAttribute('value', '1');
      await expect(bar).toHaveAttribute('max', '3');

      await checkOf(page, 'Two').check();
      await checkOf(page, 'Three').check();
      await expect(count(page)).toHaveText('All 3 done');
      await expect(bar).toHaveAttribute('value', '3');

      await checkOf(page, 'Two').uncheck();
      await expect(count(page)).toHaveText('2 of 3 done');
      await expect(rowOf(page, 'Two').locator('.step-by')).toHaveCount(0);
      expect(await notReloaded(page)).toBe(true);
    });

    test(`gate 5.03: the keyboard can tick and keeps its place (${home})`, async ({ page, server }) => {
      await openJob(page, server.baseURL, home);
      await typeSteps(page, 'One', 'Two');
      await checkOf(page, 'One').focus();
      await page.keyboard.press('Space');
      await expect(count(page)).toHaveText('1 of 2 done');
      await expect(checkOf(page, 'One')).toBeFocused();
    });

    test(`gate 5.05: a step is renamed in place — click its words, type, Enter (${home})`, async ({ page, server }) => {
      await openJob(page, server.baseURL, home);
      await typeSteps(page, 'Book the hall', 'Email the parents');

      await rowOf(page, 'Book the hall').locator('.step-words').click();
      const field = steps(page).locator('.step-rename input[type="text"]');
      await expect(field).toBeFocused();
      await expect(field).toHaveValue('Book the hall');
      await page.keyboard.type(' for Friday'); // the words are selected, so this replaces them
      await expect(field).toHaveValue(' for Friday');
      await field.fill('Book the big hall');
      await page.keyboard.press('Enter');
      await expect(words(page)).toHaveText(['Book the big hall', 'Email the parents']);
      await expect(steps(page).locator('.step-rename')).toHaveCount(0);
      expect(await notReloaded(page)).toBe(true);
    });

    test(`gate 5.05: Rename opens the same form and Escape leaves it without saving, without closing anything (${home})`, async ({ page, server }) => {
      await openJob(page, server.baseURL, home);
      await typeSteps(page, 'Book the hall');

      await steps(page).getByRole('link', { name: /^Rename/ }).click();
      const field = steps(page).locator('.step-rename input[type="text"]');
      await field.fill('Something else');
      await page.keyboard.press('Escape');
      await expect(steps(page).locator('.step-rename')).toHaveCount(0);
      await expect(words(page)).toHaveText(['Book the hall']);
      if (home === 'window') await expect(page.locator('#task-window')).toBeVisible(); // Escape only left the rename
    });

    test(`gate 5.05: a rename that is too long keeps the words and says why (${home})`, async ({ page, server }) => {
      await openJob(page, server.baseURL, home);
      await typeSteps(page, 'Book the hall');
      await rowOf(page, 'Book the hall').locator('.step-words').click();
      const long = 'y'.repeat(201);
      await steps(page).locator('.step-rename input[type="text"]').fill(long);
      await page.keyboard.press('Enter');
      await expect(steps(page).getByRole('alert')).toHaveText('A step can be up to 200 characters.');
      await expect(steps(page).locator('.step-rename input[type="text"]')).toHaveValue(long);
    });

    test(`gate 5.06: removing shows Undo at once and it puts the step back where it was (${home})`, async ({ page, server }) => {
      await openJob(page, server.baseURL, home);
      await typeSteps(page, 'One', 'Two', 'Three');

      await steps(page).getByRole('button', { name: 'Remove Two' }).click();
      await expect(words(page)).toHaveText(['One', 'Three']);
      await expect(steps(page).locator('.steps-undo')).toContainText('Removed “Two”');
      await expect(count(page)).toHaveText('0 of 2 done');

      await steps(page).getByRole('button', { name: 'Undo' }).click();
      await expect(words(page)).toHaveText(['One', 'Two', 'Three']);
      await expect(steps(page).locator('.steps-undo')).toHaveCount(0);
      expect(await notReloaded(page)).toBe(true);
    });

    test(`gate 5.06: Undo goes at the next action, and after about ten seconds (${home})`, async ({ page, server }) => {
      await page.clock.install();
      await openJob(page, server.baseURL, home);
      await typeSteps(page, 'One', 'Two');

      await steps(page).getByRole('button', { name: 'Remove One' }).click();
      await expect(steps(page).locator('.steps-undo')).toHaveCount(1);
      await page.clock.runFor(9_000);
      await expect(steps(page).locator('.steps-undo')).toHaveCount(1);
      await page.clock.runFor(2_000);
      await expect(steps(page).locator('.steps-undo')).toHaveCount(0);

      await steps(page).getByRole('button', { name: 'Remove Two' }).click();
      await expect(steps(page).locator('.steps-undo')).toHaveCount(1);
      await box(page).fill('Another');
      await steps(page).getByRole('button', { name: 'Add' }).click();
      await expect(steps(page).locator('.steps-undo')).toHaveCount(0); // the next action took it away
    });

    test(`gate 5.07: the up and down icons put steps in order, with a tooltip and a spoken name (${home})`, async ({ page, server }) => {
      await openJob(page, server.baseURL, home);
      await typeSteps(page, 'One', 'Two', 'Three');

      const up = steps(page).getByRole('button', { name: 'Move Three up' });
      await expect(up).toHaveAttribute('title', 'Move Three up');
      await expect(steps(page).getByRole('button', { name: 'Move One up' })).toBeDisabled();
      await up.click();
      await expect(words(page)).toHaveText(['One', 'Three', 'Two']);
      await expect(steps(page).getByRole('button', { name: 'Move One down' })).toBeVisible();
      await steps(page).getByRole('button', { name: 'Move One down' }).click();
      await expect(words(page)).toHaveText(['Three', 'One', 'Two']);
      // The keyboard stays on the button it pressed.
      await expect(steps(page).getByRole('button', { name: 'Move One down' })).toBeFocused();
    });

    test(`gate 5.07: dragging a step by its words reorders it (${home})`, async ({ page, server }) => {
      const { title } = await openJob(page, server.baseURL, home);
      await typeSteps(page, 'One', 'Two', 'Three');

      const source = rowOf(page, 'One').locator('.step-body');
      const target = rowOf(page, 'Three');
      await source.scrollIntoViewIfNeeded();
      const from = (await source.boundingBox())!;
      const to = (await target.boundingBox())!;
      const x = from.x + 20;
      const y = from.y + from.height / 2;
      await page.mouse.move(x, y);
      await page.mouse.down();
      await page.mouse.move(x, y + 8, { steps: 4 });
      await page.mouse.move(x, to.y + to.height * 0.8, { steps: 12 });
      await page.mouse.up();
      await expect(words(page)).toHaveText(['Two', 'Three', 'One']);
      // What the server holds is what the screen shows.
      await page.reload();
      if (home === 'window') await openIn(page, title, 'window');
      await expect(words(page)).toHaveText(['Two', 'Three', 'One']);
    });

    test(`gate 5.13: the two limits keep what was typed and say so, without a reload (${home})`, async ({ page, server }) => {
      await openJob(page, server.baseURL, home);
      await box(page).click();
      await page.keyboard.press('Enter');
      await expect(steps(page).getByRole('alert')).toHaveText('Type what the step is first.');

      const long = 'z'.repeat(201);
      await box(page).fill(long);
      await page.keyboard.press('Enter');
      await expect(steps(page).getByRole('alert')).toHaveText('A step can be up to 200 characters.');
      await expect(box(page)).toHaveValue(long);
      await expect(words(page)).toHaveCount(0);
      expect(await notReloaded(page)).toBe(true);
    });

    test(`gate 5.16: a full list of 50 steps passes the accessibility check, and so does a rename (${home})`, async ({ page, server }) => {
      const { title } = await openJob(page, server.baseURL, home);
      const jobPath = await steps(page).getAttribute('data-task-id');
      for (let i = 1; i <= 50; i++) {
        const res = await page.request.post(`${server.baseURL}/tasks/${jobPath}/steps`, {
          headers: { origin: server.baseURL },
          form: { text: `A step with some words in it, number ${i}` },
          maxRedirects: 0,
        });
        expect(res.status()).toBe(302);
      }
      await page.reload();
      if (home === 'window') await openIn(page, title, 'window');
      await expect(words(page)).toHaveCount(50);
      await checkOf(page, 'A step with some words in it, number 3').check();
      await expect(count(page)).toHaveText('1 of 50 done');
      await axeCheck(page);

      await rowOf(page, 'number 7').locator('.step-words').click();
      await expect(steps(page).locator('.step-rename input[type="text"]')).toBeFocused();
      await axeCheck(page);
    });
  });
}

test('gate 5.12: a change on another computer arrives within a minute, and never over what is being typed', async ({ page, server, browser }) => {
  await page.clock.install();
  const { title } = await openJob(page, server.baseURL, 'page');
  await typeSteps(page, 'Book the hall', 'Email the parents');
  const jobUrl = page.url();

  const other = await browser.newContext();
  const otherPage = await other.newPage();
  await signInAsNewPerson(otherPage, server.baseURL, '/tasks/board');
  await otherPage.goto(jobUrl);
  await expect(words(otherPage)).toHaveText(['Book the hall', 'Email the parents']);
  await checkOf(otherPage, 'Book the hall').check();
  await otherPage.getByPlaceholder('Add a step').fill('Order the programmes');
  await otherPage.getByPlaceholder('Add a step').press('Enter');
  await expect(words(otherPage)).toHaveCount(3);

  // While somebody is half-way through typing a step here, nothing is swapped.
  await box(page).fill('Half a thought');
  await page.clock.runFor(61_000);
  await expect(words(page)).toHaveCount(2);
  await expect(box(page)).toHaveValue('Half a thought');

  // Once the box is empty again, the other person's changes come in.
  await box(page).fill('');
  await page.clock.runFor(61_000);
  await expect(words(page)).toHaveText(['Book the hall', 'Email the parents', 'Order the programmes']);
  await expect(checkOf(page, 'Book the hall')).toBeChecked();
  expect(title).toBeTruthy();
  await other.close();
});

test('gate 5.12: clicking back into the window brings the other computer’s change in straight away', async ({ page, server, browser }) => {
  await openJob(page, server.baseURL, 'page');
  await typeSteps(page, 'Book the hall');
  const jobUrl = page.url();

  const other = await browser.newContext();
  const otherPage = await other.newPage();
  await signInAsNewPerson(otherPage, server.baseURL, '/tasks/board');
  await otherPage.goto(jobUrl);
  await checkOf(otherPage, 'Book the hall').check();
  await expect(count(otherPage)).toHaveText('All 1 done');

  await page.evaluate(() => window.dispatchEvent(new Event('focus')));
  await expect(count(page)).toHaveText('All 1 done');
  await other.close();
});

test('gate 5.09: the job page and the window show and edit the same steps, and a refresh lands somewhere whole', async ({ page, server }) => {
  const { title } = await openJob(page, server.baseURL, 'window');
  await typeSteps(page, 'Print exam papers', 'Book the hall');
  await checkOf(page, 'Book the hall').check();
  await expect(count(page)).toHaveText('1 of 2 done');
  const href = await page.locator('#task-window a', { hasText: 'Open the page' }).getAttribute('href');

  // The page shows what the window made, ticks included, and edits it the same way.
  await page.goto(new URL(href!, page.url()).href);
  await ready(page);
  await expect(words(page)).toHaveText(['Print exam papers', 'Book the hall']);
  await expect(checkOf(page, 'Book the hall')).toBeChecked();
  await expect(count(page)).toHaveText('1 of 2 done');
  await checkOf(page, 'Print exam papers').check();
  await expect(count(page)).toHaveText('All 2 done');

  // A refresh mid-edit still answers, whole.
  await rowOf(page, 'Book the hall').locator('.step-words').click();
  await page.reload();
  await ready(page);
  await expect(words(page)).toHaveText(['Print exam papers', 'Book the hall']);
  await expect(count(page)).toHaveText('All 2 done');

  // And back over the Board, the window shows the page's change.
  await page.goto(server.baseURL + '/tasks/board');
  await ready(page);
  await openIn(page, title, 'window');
  await expect(count(page)).toHaveText('All 2 done');
});

test('the window does not ask "Leave without saving?" because of what was typed in Steps', async ({ page, server }) => {
  await openJob(page, server.baseURL, 'window');
  await box(page).fill('A half-typed step');
  let asked = false;
  page.on('dialog', async (d) => {
    asked = true;
    await d.dismiss();
  });
  await page.getByRole('button', { name: 'Close' }).click();
  await expect(page.locator('#task-window')).toBeHidden();
  expect(asked).toBe(false);
});

test('the window fits its screen and the page has no sideways scroll with steps in it', async ({ page, server }) => {
  await openJob(page, server.baseURL, 'window');
  await typeSteps(page, 'A rather long step that goes on and on to prove that words wrap inside the window instead of pushing it wider than the screen');
  for (const [width, height] of [
    [1024, 700],
    [1920, 1080],
  ]) {
    await page.setViewportSize({ width, height });
    const fits = await page.locator('#task-window').evaluate((el) => {
      const r = el.getBoundingClientRect();
      return r.right <= window.innerWidth && r.bottom <= window.innerHeight;
    });
    expect(fits, `${width}x${height}`).toBe(true);
  }
  await page.getByRole('link', { name: 'Open the page' }).click();
  await ready(page);
  await expectNoSideScroll(page);
});
