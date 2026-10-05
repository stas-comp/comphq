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
    if (await waitForChosen(page, taskId, 3000)) return;
    await page.mouse.up();
  }
  throw new Error('beginDrag: the drag never started after retries');
}

/** Moves the pointer in steps, giving the page real time between waypoints. */
export async function dragPointerTo(page: Page, x: number, y: number): Promise<void> {
  await page.mouse.move(x, y, { steps: 10 });
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
    await page.mouse.move(x + (i % 2), y);
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
 * has been brought into view.
 */
export async function dragCardTo(
  page: Page,
  card: Locator,
  target: () => Promise<{ x: number; y: number }>,
  endpoint?: RegExp,
): Promise<number> {
  for (let attempt = 1; attempt <= 4; attempt++) {
    await beginDrag(page, card);
    const { x, y } = await target();
    await dragPointerTo(page, x, y);
    const status = await endDrag(page, endpoint);
    if (status !== 0) return status;
  }
  throw new Error('dragCardTo: the drop never registered after retries');
}
