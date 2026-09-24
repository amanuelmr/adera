// Lets the API-auth middleware (src/auth/middleware.ts) tell AuthProvider to
// drop to signed-out state after a background refresh fails (theft/family
// revocation), without either module importing the other.
type Listener = () => void;

const listeners = new Set<Listener>();

export function onForcedSignOut(listener: Listener): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function emitForcedSignOut(): void {
  for (const listener of listeners) listener();
}
