import { File, Paths } from 'expo-file-system';
import * as Sharing from 'expo-sharing';

import { apiClient, unwrap } from '@/api/client';

const EXPORT_FILE_NAME = 'adera-data-export.json';

/**
 * Fetches everything the server holds about the signed-in user and hands it
 * to the system share sheet as a JSON file, so they can save it or send it
 * wherever they like. The file is written to the cache directory and
 * replaced on each export. Its photo and receipt links expire after an hour.
 */
export async function shareDataExport(dialogTitle: string): Promise<void> {
  const { data } = unwrap(await apiClient.GET('/api/v1/users/me/data-export'));
  const file = new File(Paths.cache, EXPORT_FILE_NAME);
  file.create({ overwrite: true });
  file.write(JSON.stringify(data, null, 2));
  await Sharing.shareAsync(file.uri, { mimeType: 'application/json', UTI: 'public.json', dialogTitle });
}
