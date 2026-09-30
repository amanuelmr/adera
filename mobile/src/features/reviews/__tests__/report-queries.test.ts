import en from '@/lib/locales/en.json';
import am from '@/lib/locales/am.json';
import { REPORT_REASONS } from '../report-queries';

jest.mock('@/api/env', () => ({ apiBaseUrl: 'http://api.test' }));

// Must match internal/moderation/moderation.go ReportReasons and the
// ReportCreate enum; a missing one would be unreportable from the app.
it('offers every reason the API accepts, each labeled in both languages', () => {
  expect([...REPORT_REASONS].sort()).toEqual(
    [
      'conflict_of_interest',
      'duplicate',
      'fake_experience',
      'harassment',
      'hate_speech',
      'irrelevant',
      'manipulated_evidence',
      'personal_information',
      'spam',
      'unsupported_accusation',
    ].sort()
  );
  for (const reason of REPORT_REASONS) {
    expect(en.report.reasons).toHaveProperty(reason);
    expect(am.report.reasons).toHaveProperty(reason);
  }
});
