import * as Linking from 'expo-linking';

import { WEB_BASE_URL } from '@/features/share/share';

// The privacy notice lives on the web (internal/pages, /privacy) so the same
// text is what the app links to, what search engines index, and what the
// server's privacy_notice_version dates.
export function privacyNoticeUrl(language: string): string {
  return `${WEB_BASE_URL}/privacy?lang=${language === 'am' ? 'am' : 'en'}`;
}

export function openPrivacyNotice(language: string): Promise<true> {
  return Linking.openURL(privacyNoticeUrl(language));
}
