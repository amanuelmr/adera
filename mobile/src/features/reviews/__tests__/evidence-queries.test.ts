import { countsTowardEvidenceLimit } from '../evidence-queries';

jest.mock('@/api/env', () => ({ apiBaseUrl: 'http://api.test' }));

it('matches the server: rejected evidence frees a slot, staged and pending ones do not', () => {
  const items = [{ status: 'accepted' }, { status: 'submitted' }, { status: 'staged' }, { status: 'rejected' }];
  expect(items.filter(countsTowardEvidenceLimit)).toHaveLength(3);
});
