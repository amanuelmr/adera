import * as Haptics from 'expo-haptics';
import { Platform } from 'react-native';

// Light physical feedback for choices and outcomes. Android uses its own
// haptic primitives (closer to iOS, and gentler than the vibration motor
// the cross-platform calls fall back to there). Fire-and-forget: a device
// without haptics simply does nothing.

/** A choice changed: a chip, a star, a tab. */
export function selectionFeedback(): void {
  const done =
    Platform.OS === 'android'
      ? Haptics.performAndroidHapticsAsync(Haptics.AndroidHaptics.Segment_Tick)
      : Haptics.selectionAsync();
  done.catch(() => undefined);
}

/** Something the user did succeeded: a review posted, a vote recorded. */
export function successFeedback(): void {
  const done =
    Platform.OS === 'android'
      ? Haptics.performAndroidHapticsAsync(Haptics.AndroidHaptics.Confirm)
      : Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
  done.catch(() => undefined);
}
