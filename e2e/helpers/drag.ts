import type { Locator, Page } from '@playwright/test';

// Dragging with SortableJS under a synthetic mouse (D-41): real pauses
// between waypoints, a wait for the library's own "drag started" class
// instead of a guessed sleep, and a retried gesture. Every Tasks drag test
// goes through here.

async function waitForChosen(page: Page, taskId: string, timeout: number): Promise<boolean> {
  try {
    await page.waitForFunction(
      (id) => document.querySelector(`[data-task-id="${id}"]`)?.classList.contains('sortable-chosen'),
      taskId,
      { timeout },
    );
    return true;
  } catch {
    return false;
  }
}

/** Presses on a card's title and moves just far enough that the drag has begun. */
export async function beginDrag(page: Page, card: Locator): Promise<void> {
  const taskId = await card.getAttribute('data-task-id');
  if (!taskId) throw new Error('beginDrag: not a card');
  for (let attempt = 1; attempt <= 4; attempt++) {
    await card.scrollIntoViewIfNeeded();
    const box = await card.locator('.task-card-title').first().boundingBox();
    if (!box) throw new Error('beginDrag: card has no title box');
    const x = box.x + box.width / 2;
    const y = box.y + box.height / 2;
    await page.mouse.move(x, y);
    await page.mouse.down();
    await page.mouse.move(x, y + 10, { steps: 5 });
    pointer.set(page, { x, y: y + 10 });
    if (await waitForChosen(page, taskId, 3000)) return;
    await page.mouse.up();
  }
  throw new Error('beginDrag: the drag never started after retries');
}

// Where each page's synthetic pointer is (Playwright doesn't say).
const pointer = new WeakMap<Page, { x: number; y: number }>();

/**
 * Moves the pointer to (x, y) in steps, giving the page real time between
 * them. SortableJS works out where the card is over through throttled
 * handlers (every 30-50 ms), so one quick burst of moves can end with it
 * still believing the card is over an earlier spot. Walk there with real
 * pauses, then settle with two tiny moves on the spot.
 */
export async function dragPointerTo(page: Page, x: number, y: number): Promise<void> {
  const from = pointer.get(page) ?? { x, y };
  const STEPS = 10;
  for (let i = 1; i <= STEPS; i++) {
    await page.mouse.move(from.x + ((x - from.x) * i) / STEPS, from.y + ((y - from.y) * i) / STEPS);
    await page.waitForTimeout(70);
  }
  await page.mouse.move(x + 1, y);
  await page.waitForTimeout(60);
  await page.mouse.move(x, y);
  await page.waitForTimeout(60);
  pointer.set(page, { x, y });
}

/**
 * Holds the pointer near (x, y) the way a hand does, with tiny movements and
 * real pauses, until `done()` is true or the time is up. SortableJS reads the
 * pointer through a 30 ms throttle, so a perfectly still synthetic pointer
 * that arrived in one quick burst can leave it with a stale position.
 */
export async function holdPointer(
  page: Page,
  x: number,
  y: number,
  done: () => Promise<boolean>,
  timeoutMs = 20_000,
): Promise<void> {
  const deadline = Date.now() + timeoutMs;
  for (let i = 0; Date.now() < deadline; i++) {
    // Tiny sideways steps, and every few turns a short dip back down and up
    // again, so the library sees the pointer enter the target afresh.
    await page.mouse.move(x + (i % 2), y + (i % 6 === 5 ? 20 : 0));
    pointer.set(page, { x, y });
    if (await done()) return;
    await page.waitForTimeout(40);
  }
  throw new Error('holdPointer: the page never reacted');
}

/**
 * Releases the button and returns the status of the move request the drop
 * sends, or 0 if none was sent.
 */
export async function endDrag(page: Page, endpoint: RegExp = /\/tasks\/\d+\/move$/): Promise<number> {
  const moved = page.waitForResponse(
    (res) => endpoint.test(new URL(res.url()).pathname) && res.request().method() === 'POST',
    { timeout: 5000 },
  );
  await page.mouse.up();
  try {
    return (await moved).status();
  } catch {
    return 0;
  }
}

/**
 * A whole drag: from a card's title to a point, retried when the drop never
 * registered. `target` is evaluated fresh on every attempt, after the card
 * has been brought into view. `settled`, when given, says when the dragged
 * card has reached the place the test means (SortableJS moves it as it
 * decides where the pointer is); the pointer is held there, with tiny
 * movements, until it is true, and only then released. If it never is, the
 * gesture is let go and started again, unless `restart` is false (a drag whose
 * drop changes what the card is, such as an idea becoming a job, can't be
 * repeated).
 */
export async function dragCardTo(
  page: Page,
  card: Locator,
  target: () => Promise<{ x: number; y: number }>,
  opts: { endpoint?: RegExp; settled?: () => Promise<boolean>; restart?: boolean } = {},
): Promise<number> {
  for (let attempt = 1; attempt <= 4; attempt++) {
    await beginDrag(page, card);
    const { x, y } = await target();
    await dragPointerTo(page, x, y);
    let settled = true;
    if (opts.settled) {
      // The page can shift under a drag (the card being carried leaves its
      // place, and SortableJS may scroll), so re-measure the target on every
      // turn instead of trusting where it was when the drag began.
      settled = false;
      const deadline = Date.now() + (opts.restart === false ? 10_000 : 4000);
      for (let i = 0; Date.now() < deadline && !settled; i++) {
        const t = await target();
        await page.mouse.move(t.x + (i % 2), t.y);
        pointer.set(page, t);
        settled = await opts.settled();
        if (settled) {
          // Stay put a moment and make sure it stays that way before letting go.
          await page.waitForTimeout(150);
          settled = await opts.settled();
        }
        if (!settled) await page.waitForTimeout(40);
      }
      if (!settled && attempt < 4 && opts.restart !== false) {
        // The library never put the card where the test means (a busy
        // machine can starve its timers). Let go and start the gesture again.
        await endDrag(page, opts.endpoint);
        await page.waitForTimeout(300);
        continue;
      }
    }
    const status = await endDrag(page, opts.endpoint);
    if (status !== 0) return status;
  }
  throw new Error('dragCardTo: the drop never registered after retries');
}
