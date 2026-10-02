import { resolveNotificationRoute } from '../resolve-navigation';

const mockGet = jest.fn();
jest.mock('@/api/client', () => ({
  apiClient: { GET: (...args: unknown[]) => mockGet(...args) },
  unwrap: (r: { data: unknown }) => r.data,
}));

function notification(overrides: object) {
  return {
    id: 'n1',
    event_type: 'review.hidden',
    subject_type: 'review',
    subject_id: 'review-1',
    data: {},
    created_at: '2026-10-01T00:00:00Z',
    ...overrides,
  } as never;
}

beforeEach(() => mockGet.mockReset());

it('opens the evidence screen for an evidence decision', async () => {
  const n = notification({ event_type: 'evidence.accepted', subject_type: 'evidence', subject_id: 'ev-1', data: { review_id: 'r-9' } });
  expect(await resolveNotificationRoute(n)).toBe('/review/r-9/evidence');
});

it('opens Your businesses only for an approved claim', async () => {
  expect(await resolveNotificationRoute(notification({ event_type: 'claim.approved', subject_type: 'claim' }))).toBe('/businesses');
  expect(await resolveNotificationRoute(notification({ event_type: 'claim.rejected', subject_type: 'claim' }))).toBeUndefined();
});

it("resolves a response notification to the review's place", async () => {
  mockGet.mockResolvedValue({ data: { data: { target_id: 't-7' } } });
  const n = notification({ event_type: 'response.created', subject_type: 'response', subject_id: 'resp-1', data: { review_id: 'r-1' } });
  expect(await resolveNotificationRoute(n)).toBe('/target/t-7');
});

it('goes nowhere when the review lookup fails', async () => {
  mockGet.mockRejectedValue(new TypeError('Network request failed'));
  expect(await resolveNotificationRoute(notification({}))).toBeUndefined();
});

it('goes nowhere for a report outcome', async () => {
  expect(await resolveNotificationRoute(notification({ event_type: 'report.resolved', subject_type: 'report', data: {} }))).toBeUndefined();
});
