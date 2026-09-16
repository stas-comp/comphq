// Every page type gets an axe test and a screenshot entry, in the same
// task that creates the page (SPEC §2.8). Add one row here per new page
// type; both e2e/a11y/pages.spec.ts and e2e/screens/pages.spec.ts read it.
// needsPerson: false for pages meant to be reached without a person cookie
// (the picker itself); true for everything behind the identity middleware.
export type RegisteredPage = { name: string; path: string; needsPerson: boolean };

export const pages: RegisteredPage[] = [
  { name: 'who', path: '/who', needsPerson: false },
  { name: 'frame', path: '/', needsPerson: true },
];
