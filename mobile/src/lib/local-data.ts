import AsyncStorage from '@react-native-async-storage/async-storage';
import { Directory, Paths } from 'expo-file-system';

import { DRAFT_KEY_PREFIX } from '@/features/reviews/drafts';
import { clearQueue } from '@/features/reviews/offline-queue';
import { PERSISTED_PHOTO_PREFIX } from '@/features/reviews/photos';

/**
 * Removes what an account leaves on the device besides its tokens and the
 * query cache: unfinished review drafts, reviews and votes queued while
 * offline, and the photos both hold. Run when someone signs out or erases
 * their account — after that the phone may be someone else's. (A forced
 * sign-out, when a session merely expires, keeps them: the same person is
 * about to sign back in, and everything is keyed to their account anyway.)
 *
 * Best-effort throughout: a failure here mustn't keep anyone signed in.
 */
export async function wipeLocalAccountData(): Promise<void> {
  await clearQueue().catch(() => undefined);
  try {
    const keys = await AsyncStorage.getAllKeys();
    await AsyncStorage.multiRemove(keys.filter((key) => key.startsWith(DRAFT_KEY_PREFIX)));
  } catch {
    // Storage unreadable: nothing more we can do.
  }
  try {
    for (const item of new Directory(Paths.document).list()) {
      if (item.name.startsWith(PERSISTED_PHOTO_PREFIX)) item.delete();
    }
  } catch {
    // Directory missing or unreadable: no photos to remove.
  }
}
