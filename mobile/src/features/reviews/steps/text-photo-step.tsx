import { useState } from 'react';
import { Image, Pressable, StyleSheet, TextInput, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { Spacing } from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';
import { deletePersistedPhoto, pickAndCompressPhoto } from '../photos';
import type { PickedPhoto } from '../types';

const MAX_PHOTOS = 5;
const MIN_BODY_LENGTH = 20;

export function TextPhotoStep({
  title,
  body,
  photos,
  onChangeTitle,
  onChangeBody,
  onChangePhotos,
}: {
  title: string;
  body: string;
  photos: PickedPhoto[];
  onChangeTitle: (title: string) => void;
  onChangeBody: (body: string) => void;
  onChangePhotos: (photos: PickedPhoto[]) => void;
}) {
  const theme = useTheme();
  const [photoError, setPhotoError] = useState<string>();

  async function addPhoto() {
    setPhotoError(undefined);
    const result = await pickAndCompressPhoto();
    if ('error' in result) setPhotoError(result.error);
    else if ('uri' in result) onChangePhotos([...photos, { uri: result.uri }]);
  }

  return (
    <View style={styles.container}>
      <ThemedText type="subtitle">Tell us more</ThemedText>

      <TextInput
        value={title}
        onChangeText={onChangeTitle}
        placeholder="Title (optional)"
        placeholderTextColor={theme.textSecondary}
        maxLength={120}
        style={[styles.titleInput, { color: theme.text, backgroundColor: theme.backgroundElement }]}
      />

      <TextInput
        value={body}
        onChangeText={onChangeBody}
        placeholder="What happened? What should other people know?"
        placeholderTextColor={theme.textSecondary}
        multiline
        maxLength={5000}
        style={[styles.bodyInput, { color: theme.text, backgroundColor: theme.backgroundElement }]}
      />
      <ThemedText type="small" themeColor="textSecondary">
        {body.length < MIN_BODY_LENGTH ? `${MIN_BODY_LENGTH - body.length} more characters needed` : `${body.length}/5000`}
      </ThemedText>

      <View style={styles.photosRow}>
        {photos.map((photo, index) => (
          <View key={photo.uri} style={styles.photoWrapper}>
            <Image source={{ uri: photo.uri }} style={styles.photo} />
            <Pressable
              onPress={() => {
                deletePersistedPhoto(photo.uri);
                onChangePhotos(photos.filter((_, i) => i !== index));
              }}
              accessibilityRole="button"
              accessibilityLabel="Remove photo"
              style={[styles.removeButton, { backgroundColor: theme.background }]}>
              <ThemedText type="smallBold">×</ThemedText>
            </Pressable>
          </View>
        ))}
        {photos.length < MAX_PHOTOS ? (
          <Pressable
            onPress={addPhoto}
            accessibilityRole="button"
            accessibilityLabel="Add photo"
            style={[styles.addPhoto, { backgroundColor: theme.backgroundElement }]}>
            <ThemedText type="title">+</ThemedText>
          </Pressable>
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
