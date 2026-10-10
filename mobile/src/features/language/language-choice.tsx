import { useState } from 'react';
import { Modal, Pressable, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { Spacing } from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';
import { setAppLanguage, useNeedsLanguageChoice, type AppLanguage } from '@/lib/i18n';

// Each option is written in its own language, and the heading in both: the
// reader may not read the language the device guessed. Plain Text rather
// than ThemedText, which would set the Amharic option in the Latin font
// when the guess was English.
const OPTIONS: { language: AppLanguage; label: string }[] = [
  { language: 'am', label: 'አማርኛ' },
  { language: 'en', label: 'English' },
];

/**
 * Shown once, over the app, on first launch. A modal rather than a
 * replacement screen, so a link that opened the app is still on screen
 * underneath once a language is picked.
 */
export function LanguageChoice() {
  const needsChoice = useNeedsLanguageChoice();
  const theme = useTheme();
  const [saving, setSaving] = useState<AppLanguage>();

  if (!needsChoice) return null;

  async function choose(language: AppLanguage) {
    setSaving(language);
    try {
      await setAppLanguage(language);
    } finally {
      setSaving(undefined);
    }
  }

  return (
    <Modal visible animationType="fade" onRequestClose={() => undefined} statusBarTranslucent>
      <SafeAreaView style={[styles.container, { backgroundColor: theme.background }]}>
        <View style={styles.heading}>
          <Text style={[styles.title, { color: theme.text }]} accessibilityRole="header">
            ቋንቋ ይምረጡ
          </Text>
          <Text style={[styles.title, { color: theme.text }]} accessibilityRole="header">
            Choose your language
          </Text>
        </View>
        <View style={styles.options}>
          {OPTIONS.map((option) => (
            <Pressable
              key={option.language}
              onPress={() => choose(option.language)}
              disabled={saving !== undefined}
              accessibilityRole="button"
              accessibilityLanguage={option.language}
              accessibilityState={{ busy: saving === option.language }}
              style={({ pressed }) => [
                styles.option,
                { backgroundColor: theme.backgroundElement, opacity: pressed ? 0.8 : 1 },
              ]}>
              <Text style={[styles.optionLabel, { color: theme.text }]}>{option.label}</Text>
            </Pressable>
          ))}
        </View>
      </SafeAreaView>
    </Modal>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    justifyContent: 'center',
    padding: Spacing.four,
    gap: Spacing.five,
  },
  heading: {
    gap: Spacing.two,
  },
  title: {
    fontSize: 24,
    lineHeight: 39,
    fontWeight: 600,
    textAlign: 'center',
  },
  options: {
    gap: Spacing.three,
  },
  option: {
    minHeight: 56,
    borderRadius: Spacing.three,
    alignItems: 'center',
    justifyContent: 'center',
  },
  optionLabel: {
    fontSize: 20,
    lineHeight: 32,
    fontWeight: 600,
  },
});
