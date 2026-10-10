import { Image } from 'expo-image';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { FlatList, Modal, Pressable, StyleSheet, Text, View, useWindowDimensions } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { Spacing } from '@/constants/theme';

export type ViewerPhoto = { id: string; url: string; thumbUrl?: string };

// Full-screen, swipeable photo viewer. The thumbnail is used as the
// placeholder, so the already-downloaded small image shows instantly while
// the full-size one loads (and both stay in expo-image's disk cache).
export function PhotoViewer({
  photos,
  initialIndex,
  onClose,
}: {
  photos: ViewerPhoto[];
  initialIndex: number;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const { width, height } = useWindowDimensions();
  const insets = useSafeAreaInsets();
  const [index, setIndex] = useState(initialIndex);

  return (
    <Modal visible animationType="fade" onRequestClose={onClose} statusBarTranslucent supportedOrientations={['portrait']}>
      <View style={styles.backdrop}>
        <FlatList
          data={photos}
          keyExtractor={(item) => item.id}
          horizontal
          pagingEnabled
          showsHorizontalScrollIndicator={false}
          initialScrollIndex={initialIndex}
          getItemLayout={(_, i) => ({ length: width, offset: width * i, index: i })}
          onMomentumScrollEnd={(event) => setIndex(Math.round(event.nativeEvent.contentOffset.x / width))}
          renderItem={({ item, index: i }) => (
            <Image
              source={{ uri: item.url }}
              placeholder={item.thumbUrl ? { uri: item.thumbUrl } : undefined}
              placeholderContentFit="contain"
              contentFit="contain"
              transition={150}
              accessibilityLabel={t('photos.position', { index: i + 1, count: photos.length })}
              style={{ width, height }}
            />
          )}
        />

        <View style={[styles.topBar, { paddingTop: insets.top + Spacing.two }]} pointerEvents="box-none">
          {photos.length > 1 ? (
            <Text style={styles.counter} accessibilityLiveRegion="polite">
              {t('photos.position', { index: index + 1, count: photos.length })}
            </Text>
          ) : (
            <View />
          )}
          <Pressable
            onPress={onClose}
            accessibilityRole="button"
            accessibilityLabel={t('photos.close')}
            hitSlop={12}
            style={styles.closeButton}>
            <Text style={styles.closeText}>×</Text>
          </Pressable>
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: {
    flex: 1,
    backgroundColor: '#000',
  },
  topBar: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingHorizontal: Spacing.three,
  },
  counter: {
    color: '#fff',
    fontSize: 14,
  },
  closeButton: {
    width: 44,
    height: 44,
    borderRadius: 22,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: 'rgba(0,0,0,0.5)',
  },
  closeText: {
    color: '#fff',
    fontSize: 28,
    lineHeight: 30,
  },
});
