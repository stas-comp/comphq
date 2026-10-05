import { expect, test } from '../helpers/fixtures';
import { generatePNG } from '../helpers/images';
import { expectNoSideScroll } from '../helpers/no-side-scroll';
import { pages } from '../helpers/page-registry';
import { signInAsNewPerson } from '../helpers/people';
import { decodePNG, hexToRgb, sameColour } from '../helpers/pixels';
import { ready } from '../helpers/ready';
import { openNewTask } from '../helpers/tasks';
import { uniqueName } from '../helpers/unique-name';

type Page = import('@playwright/test').Page;

// SPEC B13.7, gates 7.50–7.52, D-97: no blank strip beside or below any
// page. The cause named in B13.7 is that Team's hidden labels stretch the
// whole document sideways, so the page itself can be swiped left.

// Ten people, each with a To do job, so every lane has a card with its
// (visually hidden) "Assign … to" label.
async function seedTeam(page: Page, baseURL: string, people = 10): Promise<void> {
  const names: string[] = [];
  for (let i = 0; i < people; i++) {
    names.push(await signInAsNewPerson(page, baseURL, '/tasks/board'));
  }
  for (const name of names) {
    await openNewTask(page);
    await page.fill('#new-task-title', uniqueName('Toner'));
    await page.selectOption('#new-task-stage', 'todo');
    await page.selectOption('#new-task-people', { label: name });
    await page.click('.add-task-form button[type="submit"]');
    await ready(page);
  }
}

test('7.50 Team with ten people: the page as a whole never scrolls sideways', async ({ page, server }) => {
  test.setTimeout(120_000);
  await seedTeam(page, server.baseURL);
  await page.goto(server.baseURL + '/tasks/team');
  await ready(page);
  for (const size of [
    { width: 1024, height: 700 },
    { width: 1920, height: 1080 },
  ]) {
    await page.setViewportSize(size);
    const m = await page.evaluate(() => ({
      client: document.documentElement.clientWidth,
      root: document.documentElement.scrollWidth,
      body: document.body.scrollWidth,
      lanes: document.querySelectorAll('.team-lane').length,
    }));
    expect(m.lanes).toBeGreaterThanOrEqual(10);
    expect(m.root, `documentElement is wider than the window at ${size.width}`).toBeLessThanOrEqual(m.client);
    expect(m.body).toBeLessThanOrEqual(m.client);
  }
  await expectNoSideScroll(page);
});

test('7.50 A sideways swipe over the Team lanes moves only the lanes', async ({ page, server }) => {
  test.setTimeout(120_000);
  await seedTeam(page, server.baseURL);
  await page.setViewportSize({ width: 1024, height: 700 });
  await page.goto(server.baseURL + '/tasks/team');
  await ready(page);

  const lanes = page.locator('.team-lanes');
  const box = (await lanes.boundingBox())!;
  const widths = await lanes.evaluate((el) => ({ scroll: el.scrollWidth, client: el.clientWidth }));
  expect(widths.scroll, 'ten lanes should overflow the lanes area').toBeGreaterThan(widths.client);

  await page.mouse.move(box.x + box.width / 2, box.y + 60);
  await page.mouse.wheel(500, 0);
  await expect.poll(() => lanes.evaluate((el) => el.scrollLeft)).toBeGreaterThan(0);
  expect(await page.evaluate(() => window.scrollX)).toBe(0);

  // Shift + the ordinary wheel is the same gesture on a mouse.
  const before = await lanes.evaluate((el) => el.scrollLeft);
  await page.keyboard.down('Shift');
  await page.mouse.wheel(0, 300);
  await page.keyboard.up('Shift');
  await expect.poll(() => lanes.evaluate((el) => el.scrollLeft)).toBeGreaterThan(before);
  expect(await page.evaluate(() => window.scrollX)).toBe(0);

  // The frame stays where it is.
  const sidebar = (await page.locator('.sidebar').boundingBox())!;
  const topbar = (await page.locator('.topbar').boundingBox())!;
  expect(sidebar.x).toBe(0);
  expect(topbar.y).toBe(0);
});

// Gate 7.51 on the page that used to fail: Team with ten people.
test('7.51 no blank strip at the corners: Team with ten people', async ({ page, server }) => {
  test.setTimeout(120_000);
  await seedTeam(page, server.baseURL);
  await page.goto(server.baseURL + '/tasks/team');
  await ready(page);
  for (const size of [
    { width: 1024, height: 700 },
    { width: 1920, height: 1080 },
  ]) {
    await page.setViewportSize(size);
    await page.evaluate(() => {
      window.scrollTo(document.documentElement.scrollWidth, document.documentElement.scrollHeight);
    });
    await page.waitForTimeout(50);
    const tokens = await page.evaluate(() => {
      const cs = getComputedStyle(document.documentElement);
      return { paper: cs.getPropertyValue('--color-paper'), navy: cs.getPropertyValue('--color-ink-navy') };
    });
    const allowed = [hexToRgb(tokens.paper), hexToRgb(tokens.navy)];
    const shot = decodePNG(await page.screenshot());
    for (const [x, y] of [
      [0, 0],
      [shot.width - 1, 0],
      [0, shot.height - 1],
      [shot.width - 1, shot.height - 1],
    ]) {
      const px = shot.at(x, y);
      expect(
        allowed.some((c) => sameColour(c, px)),
        `corner (${x},${y}) at ${size.width}x${size.height} is rgb(${px.r},${px.g},${px.b}), not the paper or the navy`,
      ).toBe(true);
    }
  }
});

// Gate 7.51: scrolled as far down and right as the page allows, all four
// corners of the window are the paper or the navy sidebar, never white.
for (const p of pages) {
  test(`7.51 no blank strip at the corners: ${p.name}`, async ({ page, server }) => {
    if (p.needsPerson) {
      await signInAsNewPerson(page, server.baseURL, p.path);
    } else {
      await page.goto(server.baseURL + p.path);
    }
    await ready(page);
    for (const size of [
      { width: 1024, height: 700 },
      { width: 1920, height: 1080 },
    ]) {
      await page.setViewportSize(size);
      await page.evaluate(() => {
        window.scrollTo(document.documentElement.scrollWidth, document.documentElement.scrollHeight);
      });
      await page.waitForTimeout(50); // let the scroll and any sticky bars settle
      const tokens = await page.evaluate(() => {
        const cs = getComputedStyle(document.documentElement);
        return {
          paper: cs.getPropertyValue('--color-paper'),
          navy: cs.getPropertyValue('--color-ink-navy'),
        };
      });
      const allowed = [hexToRgb(tokens.paper), hexToRgb(tokens.navy)];
      const shot = decodePNG(await page.screenshot());
      const corners = [
        [0, 0],
        [shot.width - 1, 0],
        [0, shot.height - 1],
        [shot.width - 1, shot.height - 1],
      ];
      for (const [x, y] of corners) {
        const px = shot.at(x, y);
        const label = `corner (${x},${y}) of ${p.name} at ${size.width}x${size.height} is rgb(${px.r},${px.g},${px.b})`;
        expect(px.r === 255 && px.g === 255 && px.b === 255, `${label}: white`).toBe(false);
        expect(
          allowed.some((c) => sameColour(c, px)),
          `${label}: not the paper or the navy`,
        ).toBe(true);
      }
    }
  });
}

// Gate 7.52: a picture wider than the article is shrunk to fit, on the
// article page and in the editor.
test('7.52 a 3,000 pixel wide picture fits on the article page and in the editor', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb/new');
  const upload = await page.request.post(server.baseURL + '/kb/images', {
    data: generatePNG(3000, 60),
    headers: { origin: server.baseURL, 'content-type': 'application/octet-stream' },
  });
  expect(upload.ok()).toBe(true);
  const { src } = (await upload.json()) as { src: string };

  const category = uniqueName('Wide');
  const made = await page.request.post(server.baseURL + '/kb/categories', {
    form: { name: category },
    headers: { origin: server.baseURL },
  });
  expect(made.ok()).toBe(true);
  await page.goto(server.baseURL + '/kb/new');
  await ready(page);
  const categoryID = await page
    .locator('select[name="category_id"] option', { hasText: category })
    .getAttribute('value');
  expect(categoryID).toBeTruthy();
  const title = uniqueName('Wide picture');
  const saved = await page.request.post(server.baseURL + '/kb/articles', {
    form: { title, category_id: categoryID, body_html: `<p>Before</p><img src="${src}" alt="wide"><p>After</p>` },
    headers: { origin: server.baseURL },
  });
  expect(saved.ok()).toBe(true);
  const articleURL = saved.url();
  expect(articleURL).toMatch(/\/kb\/articles\/\d+/);

  for (const size of [
    { width: 1024, height: 700 },
    { width: 1920, height: 1080 },
  ]) {
    await page.setViewportSize(size);
    await page.goto(articleURL);
    await ready(page);
    const article = await page.evaluate(() => {
      const img = document.querySelector('.kb-article-body img') as HTMLImageElement;
      const body = document.querySelector('.kb-article-body') as HTMLElement;
      return { img: img.getBoundingClientRect().width, body: body.getBoundingClientRect().width, naturalWidth: img.naturalWidth };
    });
    expect(article.naturalWidth).toBe(3000);
    expect(article.img, 'article picture wider than its column').toBeLessThanOrEqual(article.body + 0.5);
    await expectNoSideScroll(page);

    await page.goto(articleURL + '/edit');
    await ready(page);
    await page.waitForSelector('.ProseMirror img');
    const editor = await page.evaluate(() => {
      const img = document.querySelector('.ProseMirror img') as HTMLImageElement;
      const box = document.querySelector('.ProseMirror') as HTMLElement;
      return { img: img.getBoundingClientRect().width, box: box.getBoundingClientRect().width };
    });
    expect(editor.img, 'editor picture wider than the editor').toBeLessThanOrEqual(editor.box + 0.5);
    await expectNoSideScroll(page);
  }
});
