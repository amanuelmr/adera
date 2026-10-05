import { File, Paths } from 'expo-file-system';
import { ImageManipulator, SaveFormat } from 'expo-image-manipulator';
import * as ImagePicker from 'expo-image-picker';

// Client-side compression before upload — max ~1600px, ~0.7 quality — saves
// the uploader's own data, per docs/mobile-plan.md §7.
const MAX_DIMENSION = 1600;
const COMPRESS_QUALITY = 0.7;

/**
 * Caps the longest edge at MAX_DIMENSION, keeping aspect ratio (one side
 * given, the other derived). Portraits and tall screenshots are capped by
 * height, not left full size. Never upscales.
 */
export function downscaleFor(width: number, height: number): { width: number } | { height: number } | undefined {
  if (Math.max(width, height) <= MAX_DIMENSION) return undefined;
  return width >= height ? { width: MAX_DIMENSION } : { height: MAX_DIMENSION };
}

export type PickPhotoResult = { uri: string } | { error: 'permission' } | { canceled: true };
export type PhotoSource = 'library' | 'camera';

export async function pickAndCompressPhoto(source: PhotoSource = 'library'): Promise<PickPhotoResult> {
  const permission =
    source === 'camera'
      ? await ImagePicker.requestCameraPermissionsAsync()
      : await ImagePicker.requestMediaLibraryPermissionsAsync();
  if (!permission.granted) {
    return { error: 'permission' };
  }

  const picked =
    source === 'camera'
      ? await ImagePicker.launchCameraAsync({ quality: 1 })
      : await ImagePicker.launchImageLibraryAsync({ mediaTypes: ['images'], quality: 1 });
  const asset = picked.canceled ? undefined : picked.assets[0];
  if (!asset) return { canceled: true };
  return { uri: await compressAndPersist(asset) };
}

/**
 * Android can kill the app while the camera or gallery is open; the photo
 * the user just took is then delivered on the next launch instead. Returns
 * it (compressed and persisted like any other), or undefined.
 */
export async function recoverPendingPhoto(): Promise<string | undefined> {
  const pending = await ImagePicker.getPendingResultAsync();
  if (!pending || !('assets' in pending) || pending.canceled || !pending.assets?.[0]) return undefined;
  return compressAndPersist(pending.assets[0]);
}

async function compressAndPersist(asset: ImagePicker.ImagePickerAsset): Promise<string> {
  const context = ImageManipulator.manipulate(asset.uri);
  const resize = downscaleFor(asset.width, asset.height);
  if (resize) context.resize(resize);
  const rendered = await context.renderAsync();
  const result = await rendered.saveAsync({ format: SaveFormat.JPEG, compress: COMPRESS_QUALITY });

  // Moved out of cache into persistent storage: a queued review submission
  // or a saved draft can sit for hours, and expo-image-manipulator's output
  // lives in a cache dir the OS is free to clear at any time.
  const persisted = new File(Paths.document, `review-photo-${Date.now()}-${Math.random().toString(36).slice(2)}.jpg`);
  await new File(result.uri).copy(persisted);
  return persisted.uri;
}

/** Deletes a photo persisted by {@link pickAndCompressPhoto} once it's no longer needed. */
export function deletePersistedPhoto(uri: string): void {
  try {
    new File(uri).delete();
  } catch {
    // Already gone, or never one of ours — either way, nothing to clean up.
  }
}
