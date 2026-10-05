import { parsePushPayload } from '../payload';

// The shape internal/notifications/fcm.go sends: flat fields, plus the
// event's own fields JSON-encoded under `data`.
const fcmData = {
  event_type: 'review.received',
  subject_type: 'review',
  data: JSON.stringify({ rating: '2', target_id: 't-1', business_id: 'b-1' }),
};

it('reads the event type and its fields from direct keys', () => {
  expect(parsePushPayload(fcmData)).toEqual({
    eventType: 'review.received',
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
