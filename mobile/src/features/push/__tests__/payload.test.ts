import { isForSignedInUser, parsePushPayload } from '../payload';

// The shape internal/notifications/fcm.go sends: flat fields, plus the
// event's own fields JSON-encoded under `data`.
const fcmData = {
  user_id: 'u-1',
  event_type: 'review.received',
  subject_type: 'review',
  data: JSON.stringify({ rating: '2', target_id: 't-1', business_id: 'b-1' }),
};

it('reads the event type and its fields from direct keys', () => {
  expect(parsePushPayload(fcmData)).toEqual({
    eventType: 'review.received',
    userId: 'u-1',
    data: { rating: '2', target_id: 't-1', business_id: 'b-1' },
  });
});

it('reads them when everything arrives JSON-encoded under dataString', () => {
  expect(parsePushPayload({ dataString: JSON.stringify(fcmData) }).data.rating).toBe('2');
});

it('tolerates junk instead of throwing in the background task', () => {
  expect(parsePushPayload(undefined)).toEqual({ data: {} });
  expect(parsePushPayload({ dataString: '{bad' })).toEqual({ data: {} });
  expect(parsePushPayload({ event_type: 'x', data: '{bad' })).toEqual({ eventType: 'x', data: {} });
});

it('shows a push only to the account it was sent to', () => {
  const payload = parsePushPayload(fcmData);
  expect(isForSignedInUser(payload, 'u-1')).toBe(true);
  expect(isForSignedInUser(payload, 'u-2')).toBe(false);
  expect(isForSignedInUser(payload, null)).toBe(false);
  expect(isForSignedInUser(parsePushPayload({ event_type: 'x' }), 'u-1')).toBe(false);
});
