// Every page type gets an axe test and a screenshot entry, in the same
// task that creates the page (SPEC §2.8). Add one row here per new page
// type; both e2e/a11y/pages.spec.ts and e2e/screens/pages.spec.ts read it.
export type RegisteredPage = { name: string; path: string };

export const pages: RegisteredPage[] = [{ name: 'frame', path: '/' }];
