import { randomUUID } from 'expo-crypto';
import * as SecureStore from 'expo-secure-store';

// A random identifier for this app install, sent with new reviews as
// X-Install-ID. The API keeps only a keyed hash of it, so moderators can see
// several accounts reviewing a place from one phone (docs/moderation-policy.md,
// "Fraud signals"). Random rather than derived from the hardware, and kept
// across sign-outs: it identifies the install, not the person.
const STORAGE_KEY = 'adera.install-id';

let cached: Promise<string | undefined> | undefined;

async function loadOrCreate(): Promise<string | undefined> {
  try {
    const stored = await SecureStore.getItemAsync(STORAGE_KEY);
    if (stored) return stored;
    const created = randomUUID();
    await SecureStore.setItemAsync(STORAGE_KEY, created);
    return created;
  } catch {
    // A signal, not a requirement: a review is never held back for it.
    return undefined;
  }
}

/** This install's ID, or undefined if secure storage is unavailable. */
export function getInstallId(): Promise<string | undefined> {
  if (!cached) {
    cached = loadOrCreate().then((id) => {
      // Try again next time rather than remembering a failure.
      if (!id) cached = undefined;
      return id;
    });
  }
  return cached;
}
