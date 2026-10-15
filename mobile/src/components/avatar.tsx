import { StyleSheet, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { avatarColorFor, initialsOf } from '@/lib/avatar';

export type AvatarProps = {
  name: string;
  size?: number;
};

/** Initials on a colour taken from the name; decorative (the name is shown beside it). */
export function Avatar({ name, size = 40 }: AvatarProps) {
  return (
    <View
      importantForAccessibility="no-hide-descendants"
      accessibilityElementsHidden
      style={[styles.circle, { width: size, height: size, borderRadius: size / 2, backgroundColor: avatarColorFor(name) }]}>
      <ThemedText
        type="smallBold"
        style={{ color: '#FFFFFF', fontSize: Math.round(size * 0.38), lineHeight: Math.round(size * 0.5) }}>
        {initialsOf(name)}
      </ThemedText>
    </View>
  );
}

const styles = StyleSheet.create({
  circle: {
    alignItems: 'center',
    justifyContent: 'center',
  },
});
