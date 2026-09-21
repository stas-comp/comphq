import { expect, test } from '../helpers/fixtures';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';

// Comp HQ's own icon (PLAN P5-06, SPEC B11, gates 5.22, 5.23): every page links
// it, every address answers with an image, and the browser itself reads the
// manifest without complaint. Before v1.2 no page linked an icon at all, which
// is why a shortcut came out blank.

type LinkInfo = { rel: string; href: string; sizes: string };

async function iconLinks(page: import('@playwright/test').Page): Promise<LinkInfo[]> {
  return page.evaluate(() =>
    Array.from(document.querySelectorAll('link[rel~="icon"], link[rel="apple-touch-icon"]')).map((l) => ({
      rel: l.getAttribute('rel')!,
      href: (l as HTMLLinkElement).href,
      sizes: l.getAttribute('sizes') ?? '',
    })),
  );
}

test('gate 5.22: every icon address in the head answers with an image, and the browser decodes it at the size the head says', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  await ready(page);

  const links = await iconLinks(page);
  expect(links.length, 'the .ico, the PNG sizes and the apple touch icon').toBeGreaterThanOrEqual(5);
  expect(links.some((l) => l.href.endsWith('/comphq.ico'))).toBe(true);
  expect(links.some((l) => l.rel === 'apple-touch-icon')).toBe(true);

  for (const l of links) {
    const res = await page.request.get(l.href);
    expect(res.status(), l.href).toBe(200);
    expect(res.headers()['content-type'], l.href).toMatch(/^image\//);
  }

  // The browser's own decoding, not just ours: load each one as an image.
  const decoded = await page.evaluate(async (hrefs) => {
    return Promise.all(
      hrefs.map(
        (href) =>
          new Promise<{ href: string; w: number; h: number }>((resolve, reject) => {
            const img = new Image();
            img.onload = () => resolve({ href, w: img.naturalWidth, h: img.naturalHeight });
            img.onerror = () => reject(new Error('could not decode ' + href));
            img.src = href;
          }),
      ),
    );
  }, links.map((l) => l.href));
  for (const l of links) {
    const d = decoded.find((x) => x.href === l.href)!;
    expect(d.w, l.href).toBeGreaterThan(0);
    const single = /^(\d+)x(\d+)$/.exec(l.sizes);
    if (single && l.href.endsWith('.png')) expect(`${d.w}x${d.h}`, l.href).toBe(l.sizes);
  }
});

test('gate 5.22: /favicon.ico answers with the icon, even before anybody has said who they are', async ({ server, request }) => {
  const res = await request.get(server.baseURL + '/favicon.ico', { maxRedirects: 0 });
  expect(res.status()).toBe(200);
  expect(res.headers()['content-type']).toBe('image/x-icon');
  const body = await res.body();
  expect([...body.subarray(0, 4)]).toEqual([0, 0, 1, 0]);
});

test('gate 5.22: the picker and the 404 page link the icon too', async ({ page, server }) => {
  await page.goto(server.baseURL + '/who');
  expect((await iconLinks(page)).length).toBeGreaterThanOrEqual(5);
  await signInAsNewPerson(page, server.baseURL, '/kb');
  await page.goto(server.baseURL + '/no-such-page');
  await ready(page);
  expect((await iconLinks(page)).length).toBeGreaterThanOrEqual(5);
});

test('gate 5.23: the browser reads the manifest as Comp HQ, opening at the Briefing, in its own window, in the navy, with every icon', async ({ page, server }) => {
  await signInAsNewPerson(page, server.baseURL, '/kb');
  await ready(page);

  // Chromium's own view of the manifest: what an "install as an app" reads.
  const client = await page.context().newCDPSession(page);
  const result = (await client.send('Page.getAppManifest' as never)) as { url?: string; errors?: { message: string }[]; data?: string };
  expect(result.url, 'the page links a manifest').toContain('/static/app/manifest.webmanifest');
  expect(result.errors ?? [], 'the browser found nothing wrong with it').toEqual([]);
  const manifest = JSON.parse(result.data ?? '{}');
  expect(manifest.name).toBe('Comp HQ');
  expect(manifest.start_url).toBe('/');
  expect(manifest.display).toBe('standalone');
  expect(manifest.theme_color).toBe('#141b2d');
  expect(manifest.icons.map((i: { sizes: string }) => i.sizes)).toEqual(
    expect.arrayContaining(['16x16', '32x32', '48x48', '64x64', '128x128', '192x192', '256x256', '512x512']),
  );
  expect(manifest.icons.some((i: { purpose?: string }) => i.purpose === 'maskable')).toBe(true);

  // "/" is the Briefing (SPEC A8): what an installed app opens on.
  await page.goto(server.baseURL + manifest.start_url);
  await ready(page);
  await expect(page.locator('main h1, main .briefing-date, main .briefing-section').first()).toBeVisible();
  expect(new URL(page.url()).pathname).toMatch(/^\/(briefing)?$/);

  // The head carries the navy for the window frame.
  await expect(page.locator('meta[name="theme-color"]')).toHaveAttribute('content', '#141b2d');
});
