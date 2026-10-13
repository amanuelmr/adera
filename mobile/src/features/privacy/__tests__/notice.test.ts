import { privacyNoticeUrl } from '../notice';

jest.mock('@/features/share/share', () => ({ WEB_BASE_URL: 'https://adera.example' }));

it('links to the notice in the reader’s language', () => {
  expect(privacyNoticeUrl('am')).toBe('https://adera.example/privacy?lang=am');
  expect(privacyNoticeUrl('en')).toMatch(/\/privacy\?lang=en$/);
  expect(privacyNoticeUrl('om')).toMatch(/\/privacy\?lang=en$/); // no Oromo notice yet
});
