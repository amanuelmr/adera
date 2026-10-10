import { Image } from 'expo-image';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Pressable, StyleSheet, TextInput, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { Spacing } from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';
import { MIN_BODY_LENGTH, writingPromptFor } from '../form-steps';
import { deletePersistedPhoto, pickAndCompressPhoto, type PhotoSource } from '../photos';
import type { PickedPhoto } from '../types';

export const MAX_PHOTOS = 5;

export function TextPhotoStep({
  rating,
  title,
  body,
  photos,
  onChangeTitle,
  onChangeBody,
  onChangePhotos,
}: {
  rating?: number;
  title: string;
  body: string;
  photos: PickedPhoto[];
  onChangeTitle: (title: string) => void;
  onChangeBody: (body: string) => void;
  onChangePhotos: (photos: PickedPhoto[]) => void;
}) {
  const theme = useTheme();
  const { t } = useTranslation();
  const [photoError, setPhotoError] = useState<string>();
  const prompt = writingPromptFor(rating);

  async function addPhoto(source: PhotoSource) {
    setPhotoError(undefined);
    const result = await pickAndCompressPhoto(source);
    if ('error' in result) setPhotoError(t(source === 'camera' ? 'photos.cameraDenied' : 'photos.permissionDenied'));
    else if ('uri' in result) onChangePhotos([...photos, { uri: result.uri }]);
  }

  return (
    <View style={styles.container}>
      <ThemedText type="subtitle">{t('review.textTitle')}</ThemedText>

      <TextInput
        value={title}
        onChangeText={onChangeTitle}
        placeholder={t('review.titlePlaceholder')}
        placeholderTextColor={theme.textSecondary}
        maxLength={120}
        accessibilityLabel={t('review.titleLabel')}
        style={[styles.titleInput, { color: theme.text, backgroundColor: theme.backgroundElement }]}
      />

      <TextInput
        value={body}
        onChangeText={onChangeBody}
        placeholder={t('review.bodyPlaceholder')}
        placeholderTextColor={theme.textSecondary}
        multiline
        maxLength={5000}
        accessibilityLabel={t('review.bodyLabel')}
        accessibilityHint={prompt ? t(`review.prompt.${prompt}`) : undefined}
        style={[styles.bodyInput, { color: theme.text, backgroundColor: theme.backgroundElement }]}
      />
      {prompt ? (
        <ThemedText type="small" themeColor="textSecondary" importantForAccessibility="no">
          {t(`review.prompt.${prompt}`)}
        </ThemedText>
      ) : null}
      <ThemedText type="small" themeColor="textSecondary">
        {body.length < MIN_BODY_LENGTH
          ? t('review.charsNeeded', { count: MIN_BODY_LENGTH - body.length })
          : `${body.length}/5000`}
      </ThemedText>

      <View style={styles.photosRow}>
        {photos.map((photo, index) => (
          <View key={photo.uri} style={styles.photoWrapper}>
            <Image source={{ uri: photo.uri }} cachePolicy="none" style={styles.photo} />
            <Pressable
              onPress={() => {
                deletePersistedPhoto(photo.uri);
                onChangePhotos(photos.filter((_, i) => i !== index));
              }}
              accessibilityRole="button"
              accessibilityLabel={t('photos.remove')}
              hitSlop={11}
              style={[styles.removeButton, { backgroundColor: theme.background }]}>
              <ThemedText type="smallBold">×</ThemedText>
            </Pressable>
          </View>
        ))}
        {photos.length < MAX_PHOTOS ? (
          <>
            {/* Reviews are mostly written at the place: taking the photo
                there is the common case, not picking an old one. */}
            <Pressable
              onPress={() => addPhoto('camera')}
              accessibilityRole="button"
              accessibilityLabel={t('photos.takePhoto')}
              style={[styles.addPhoto, { backgroundColor: theme.backgroundElement }]}>
              <ThemedText type="smallBold">{t('photos.camera')}</ThemedText>
            </Pressable>
            <Pressable
              onPress={() => addPhoto('library')}
              accessibilityRole="button"
              accessibilityLabel={t('photos.add')}
              style={[styles.addPhoto, { backgroundColor: theme.backgroundElement }]}>
              <ThemedText type="smallBold">{t('photos.gallery')}</ThemedText>
            </Pressable>
          </>
        ) : null}
      </View>
      {photoError ? (
        <ThemedText type="small" style={styles.error}>
          {photoError}
        </ThemedText>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    gap: Spacing.three,
  },
  titleInput: {
    minHeight: 44,
    borderRadius: Spacing.two,
    paddingHorizontal: Spacing.three,
    fontSize: 16,
  },
  bodyInput: {
    minHeight: 140,
    borderRadius: Spacing.two,
    padding: Spacing.three,
    fontSize: 16,
    textAlignVertical: 'top',
  },
  photosRow: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: Spacing.two,
  },
  photoWrapper: {
    position: 'relative',
  },
  photo: {
    width: 72,
    height: 72,
    borderRadius: Spacing.one,
  },
  removeButton: {
    position: 'absolute',
    top: -Spacing.one,
    right: -Spacing.one,
    width: 22,
    height: 22,
    borderRadius: 11,
    alignItems: 'center',
    justifyContent: 'center',
  },
  addPhoto: {
    width: 72,
    height: 72,
    borderRadius: Spacing.one,
    alignItems: 'center',
    justifyContent: 'center',
  },
  error: {
    color: '#D64545',
  },
});
