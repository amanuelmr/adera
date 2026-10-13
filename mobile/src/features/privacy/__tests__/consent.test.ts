import { needsPrivacyConsent } from '../consent';

it('asks until the current notice is accepted', () => {
  expect(needsPrivacyConsent({ current_version: '2026-10-13' })).toBe(true); // older account, never asked
  expect(needsPrivacyConsent({ accepted_version: '2026-01-01', current_version: '2026-10-13' })).toBe(true);
  expect(needsPrivacyConsent({ accepted_version: '2026-10-13', current_version: '2026-10-13' })).toBe(false);
});
