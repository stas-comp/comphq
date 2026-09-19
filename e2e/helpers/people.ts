import type { Page } from '@playwright/test';
import { expect } from '@playwright/test';
import { uniqueName } from './unique-name';

// Every person a test signs in as is added to the shared server and stays
// there. The Team view draws a lane per person and a list of people inside
// every job, so a long run against one server (CI has a single worker)
// makes that page huge and slow for a reason no user will ever have. Tests
// therefore hand their people back when they finish: see retireTestPeople.
const signedIn: { baseURL: string; personID: string }[] = [];

/**
 * Removes (deactivates) every person the finished test signed in as, the
 * way Settings does, so a shared server keeps only the handful of people
 * the tests running right now need. Best effort: a server without the
 * test-mode route, or one already stopped, is not a test failure.
 */
export async function retireTestPeople(): Promise<void> {
  const people = signedIn.splice(0);
  await Promise.all(
    people.map(async ({ baseURL, personID }) => {
      try {
        await fetch(baseURL + '/__test/people/deactivate', {
          method: 'POST',
          headers: { origin: baseURL, 'content-type': 'application/x-www-form-urlencoded' },
          body: new URLSearchParams({ person_id: personID }).toString(),
        });
      } catch {
        /* nothing to clean up */
      }
    }),
  );
}

/**
 * Creates a fresh, uniquely named person via "My name isn't here" and
 * lands back on `path` (SPEC gate 1.04). Most tests that need any
 * identified person at all should use this rather than seeding one
 * directly, so each test's data — and its person — never collides with
 * another's (SPEC §2.8: "each test creates its own uniquely named data").
 */
export async function signInAsNewPerson(page: Page, baseURL: string, path = '/'): Promise<string> {
  const name = uniqueName('Test');
  await page.goto(baseURL + '/who?next=' + encodeURIComponent(path));
  await page.click('#add-name-link');
  await page.fill('#add-name-input', name);
  await page.click('#add-name-form button[type="submit"]');
  // The person cookie holds their id; keep it so the test can hand them back.
  await expect
    .poll(async () => (await page.context().cookies(baseURL)).find((c) => c.name === 'comphq_person')?.value)
    .toBeTruthy();
  const personID = (await page.context().cookies(baseURL)).find((c) => c.name === 'comphq_person')!.value;
  signedIn.push({ baseURL, personID });
  return name;
}
