import { test } from '../helpers/fixtures-today';
import { signInAsNewPerson } from '../helpers/people';
import { ready } from '../helpers/ready';
import { openNewTask } from '../helpers/tasks';

test.use({ viewport: { width: 1366, height: 768 }, today: '2026-09-19' });

// The registry's own "briefing" screenshot is of an empty Briefing (its
// friendly empty messages). This one is the same page as a Saturday
// morning looks with real content in all three sections: an overdue job,
// one due today, one of the signed-in person's own, events this week
// (one already under way) and a campaign whose notice has started.
test('@fresh screenshot: briefing-populated', async ({ page, server }) => {
  const me = await signInAsNewPerson(page, server.baseURL, '/');

  for (const task of [
    { title: 'Order Christmas card stock', due: '2026-09-10', people: [me] },
    { title: 'Print exam papers', due: '2026-09-19', people: [] as string[] },
    { title: 'Book the hall', due: '2026-09-23', people: [me] },
  ]) {
    await page.goto(server.baseURL + '/tasks/board');
    await ready(page);
    await openNewTask(page);
    await page.fill('#new-task-title', task.title);
    await page.selectOption('#new-task-stage', 'todo');
    await page.fill('#new-task-due-date', task.due);
    if (task.people.length) await page.selectOption('#new-task-people', task.people.map((label) => ({ label })));
    await page.click('.add-task-form button[type="submit"]');
    await ready(page);
  }

  for (const event of [
    { title: 'Exams', start_date: '2026-09-21', notes: 'Print exam papers today' },
    { title: 'Camp', start_date: '2026-09-17', end_date: '2026-09-22', start_time: '09:00', end_time: '15:00' },
    { title: 'Card campaign', start_date: '2026-10-31', notice_amount: '6' },
  ]) {
    await page.goto(server.baseURL + '/calendar/new');
    await ready(page);
    for (const [name, value] of Object.entries(event)) {
      if (name === 'notice_amount') continue;
      await page.locator(`[name="${name}"]`).fill(value);
    }
    if ('notice_amount' in event) {
      await page.fill('#event-notice-amount', event.notice_amount);
      await page.selectOption('#event-notice-unit', 'weeks');
    }
    await page.locator('.calendar-event-form button[type="submit"]').click();
    await ready(page);
  }

  await page.goto(server.baseURL + '/');
  await ready(page);
  await page.screenshot({ path: 'reports/screens/briefing-populated.png', fullPage: true });
});
