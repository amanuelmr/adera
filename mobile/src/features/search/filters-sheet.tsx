import { useState } from 'react';
import { Modal, Pressable, ScrollView, StyleSheet, View } from 'react-native';

import type { components } from '@/api/schema';
import { Button } from '@/components/button';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import { useCategories } from '@/features/targets/queries';
import { FilterChip } from './filter-chip';
import { MIN_RATING_OPTIONS, TARGET_TYPE_LABELS, type SearchFilters } from './types';

export type FiltersSheetProps = {
  visible: boolean;
  filters: SearchFilters;
  onApply: (filters: SearchFilters) => void;
  onClose: () => void;
};

// A React Native Modal rather than a gesture-driven bottom sheet library —
// no extra native dependency, at the cost of no drag-to-dismiss. Revisit if
// the plain version feels wrong on-device.
export function FiltersSheet({ visible, filters, onApply, onClose }: FiltersSheetProps) {
  const categories = useCategories();
  const [draft, setDraft] = useState(filters);

  // Resets the draft to the applied filters each time the sheet opens.
  // Adjusting state during render (React's documented pattern for "reset
  // when a prop changes") rather than in an effect, so opening the sheet
  // doesn't cost an extra render pass showing stale draft first.
  const [wasVisible, setWasVisible] = useState(visible);
  if (visible !== wasVisible) {
    setWasVisible(visible);
    if (visible) setDraft(filters);
  }

  function toggleType(type: components['schemas']['TargetType']) {
    setDraft((prev) => ({ ...prev, type: prev.type === type ? undefined : type }));
  }

  function toggleRating(rating: number) {
    setDraft((prev) => ({ ...prev, minRating: prev.minRating === rating ? undefined : rating }));
  }

  function toggleCategory(id: string) {
    setDraft((prev) => ({ ...prev, category: prev.category === id ? undefined : id }));
  }

  return (
    <Modal visible={visible} animationType="slide" transparent onRequestClose={onClose}>
      <Pressable style={styles.backdrop} onPress={onClose} accessibilityLabel="Close filters" />
      <ThemedView style={styles.sheet}>
        <ScrollView contentContainerStyle={styles.content}>
          <ThemedText type="subtitle">Filters</ThemedText>

          <ThemedText type="smallBold">Category</ThemedText>
          <View style={styles.row}>
            {(categories.data ?? []).map((category) => (
              <FilterChip
                key={category.id}
                label={category.name ?? 'Unnamed'}
                selected={draft.category === category.id}
                onPress={() => toggleCategory(category.id ?? '')}
              />
            ))}
          </View>

          <ThemedText type="smallBold">Type</ThemedText>
          <View style={styles.row}>
            {(Object.keys(TARGET_TYPE_LABELS) as (keyof typeof TARGET_TYPE_LABELS)[]).map((type) => (
              <FilterChip
                key={type}
                label={TARGET_TYPE_LABELS[type]}
                selected={draft.type === type}
                onPress={() => toggleType(type)}
              />
            ))}
          </View>

          <ThemedText type="smallBold">Minimum rating</ThemedText>
          <View style={styles.row}>
            {MIN_RATING_OPTIONS.map((rating) => (
              <FilterChip
                key={rating}
                label={`${rating}+`}
                selected={draft.minRating === rating}
                onPress={() => toggleRating(rating)}
              />
            ))}
          </View>

          <ThemedText type="smallBold">Verification</ThemedText>
          <View style={styles.row}>
            <FilterChip
              label="Verified only"
              selected={!!draft.verified}
              onPress={() => setDraft((prev) => ({ ...prev, verified: prev.verified ? undefined : true }))}
            />
          </View>
        </ScrollView>

        <View style={styles.actions}>
          <Button title="Clear all" variant="secondary" onPress={() => setDraft({})} />
          <Button title="Apply" onPress={() => onApply(draft)} />
        </View>
      </ThemedView>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: {
    flex: 1,
    backgroundColor: 'rgba(0,0,0,0.4)',
  },
  sheet: {
    borderTopLeftRadius: Spacing.four,
    borderTopRightRadius: Spacing.four,
    maxHeight: '80%',
  },
  content: {
    padding: Spacing.four,
    gap: Spacing.two,
  },
  row: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: Spacing.two,
    marginBottom: Spacing.three,
  },
  actions: {
    flexDirection: 'row',
    gap: Spacing.two,
    padding: Spacing.four,
  },
});
