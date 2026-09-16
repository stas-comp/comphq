// A name unique enough that tests never collide, even against a shared,
// already-populated server (SPEC §2.8).
export function uniqueName(prefix = 'Test'): string {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}
