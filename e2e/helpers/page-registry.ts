// Every page type gets an axe test and a screenshot entry, in the same
// task that creates the page (SPEC §2.8). Add one row here per new page
// type; e2e/a11y/pages.spec.ts, e2e/screens/pages.spec.ts and
// e2e/frame/no-side-scroll.spec.ts all read it.
// needsPerson: false for pages meant to be reached without a person cookie
// (the picker itself); true for everything behind the identity middleware.
export type RegisteredPage = { name: string; path: string; needsPerson: boolean };

export const pages: RegisteredPage[] = [
  { name: 'who', path: '/who', needsPerson: false },
  { name: 'briefing', path: '/briefing', needsPerson: true },
  { name: 'kb', path: '/kb', needsPerson: true },
  { name: 'kb-categories', path: '/kb/categories', needsPerson: true },
  { name: 'kb-new-article', path: '/kb/new', needsPerson: true },
  { name: 'kb-archived', path: '/kb/archived', needsPerson: true },
  { name: 'kb-search', path: '/kb/search', needsPerson: true },
  { name: 'tasks', path: '/tasks', needsPerson: true },
  { name: 'calendar', path: '/calendar', needsPerson: true },
  { name: 'settings-people', path: '/settings/people', needsPerson: true },
  { name: 'settings-backups', path: '/settings/backups', needsPerson: true },
  { name: 'settings-about', path: '/settings/about', needsPerson: true },
  { name: 'settings-setup', path: '/settings/setup', needsPerson: true },
  { name: '404', path: '/this-page-does-not-exist', needsPerson: true },
];
