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
  { name: 'tasks', path: '/tasks', needsPerson: true },
  { name: 'calendar', path: '/calendar', needsPerson: true },
  { name: '404', path: '/this-page-does-not-exist', needsPerson: true },
];
