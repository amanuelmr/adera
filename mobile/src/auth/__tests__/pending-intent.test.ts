import {
  INTENT_TTL_MS,
  clearPendingIntent,
  isSignInReason,
  setPendingIntent,
  takePendingIntent,
} from '../pending-intent';

beforeEach(() => clearPendingIntent());

it('hands back the pending action once', () => {
  const run = jest.fn();
  setPendingIntent(run, 0);
  expect(takePendingIntent(1000)).toBe(run);
  expect(takePendingIntent(1000)).toBeUndefined();
});

it('drops an intent older than the TTL', () => {
  setPendingIntent(jest.fn(), 0);
  expect(takePendingIntent(INTENT_TTL_MS + 1)).toBeUndefined();
});

it('a newer intent replaces an older one', () => {
  const first = jest.fn();
  const second = jest.fn();
  setPendingIntent(first, 0);
  setPendingIntent(second, 10);
  expect(takePendingIntent(20)).toBe(second);
});

it('only accepts known reasons (the reason can come from a deep link)', () => {
  expect(isSignInReason('writeReview')).toBe(true);
  expect(isSignInReason('<script>')).toBe(false);
  expect(isSignInReason(undefined)).toBe(false);
});

// The sign-in screen's cleanup defers its clear by a tick; the runner takes
// the intent synchronously. Whichever order they happen in within a commit,
// a completed sign-in keeps the intent and an abandoned one loses it.
it('a deferred clear does not cancel an intent already taken', () => {
  jest.useFakeTimers();
  const run = jest.fn();
  setPendingIntent(run);
  setTimeout(clearPendingIntent, 0); // sign-in screen unmounts
  const taken = takePendingIntent(); // runner, same commit
  jest.runAllTimers();
  expect(taken).toBe(run);
  jest.useRealTimers();
});

it('an abandoned sign-in clears the intent', () => {
  jest.useFakeTimers();
  setPendingIntent(jest.fn());
  setTimeout(clearPendingIntent, 0); // user backed out of sign-in
  jest.runAllTimers();
  expect(takePendingIntent()).toBeUndefined();
  jest.useRealTimers();
});
