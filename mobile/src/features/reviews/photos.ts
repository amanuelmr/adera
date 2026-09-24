import { ImageManipulator, SaveFormat } from 'expo-image-manipulator';
import * as ImagePicker from 'expo-image-picker';

// Client-side compression before upload — max ~1600px, ~0.7 quality — saves
// the uploader's own data, per docs/mobile-plan.md §7.
const MAX_DIMENSION = 1600;
const COMPRESS_QUALITY = 0.7;

export type PickPhotoResult = { uri: string } | { error: string } | { canceled: true };

export async function pickAndCompressPhoto(): Promise<PickPhotoResult> {
  const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
  if (!permission.granted) {
    return { error: 'Photo library access is needed to add a picture.' };
  }

  const picked = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ['images'],
    quality: 1,
  });
  const asset = picked.canceled ? undefined : picked.assets[0];
  if (!asset) return { canceled: true };

  const context = ImageManipulator.manipulate(asset.uri);
  // Only downscale — resizing a smaller image up would inflate it for no
  // quality gain.
  if (asset.width > MAX_DIMENSION) {
    context.resize({ width: MAX_DIMENSION });
  }
  const rendered = await context.renderAsync();
  const result = await rendered.saveAsync({ format: SaveFormat.JPEG, compress: COMPRESS_QUALITY });
  return { uri: result.uri };
}
