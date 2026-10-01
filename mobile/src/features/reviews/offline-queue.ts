import AsyncStorage from '@react-native-async-storage/async-storage';

import { apiClient, unwrap, ApiError, isTransientFailure } from '@/api/client';
import { getCurrentUserId } from '@/auth/storage';
import { isConnected } from '@/lib/network-status';
import { deletePersistedPhoto } from './photos';
import { createReview, invalidateReviewCaches, uploadPhotos } from './queries';
import type { ReviewFormState } from './types';

const STORAGE_KEY = 'adera.offline-queue.v1';

export type ReviewSubmissionJob = {
  type: 'review-submission';
  id: string;
  /** Whoever was signed in when this was queued — see processQueue(). */
  userId: string;
  targetId: string;
  idempotencyKey: string;
  form: ReviewFormState;
  /** Set once the review itself is created — retries then only resume photos. */
  reviewId?: string;
  /** Photos not yet uploaded (starts as all of them; shrinks as photos succeed or are dropped). */
  pendingPhotoUris: string[];
};

export type HelpfulVoteJob = {
  type: 'helpful-vote';
  id: string;
  userId: string;
  reviewId: string;
  voted: boolean;
};

/**
 * Set when the server rejected the job outright. The job is kept (not
 * silently dropped) so the user, who was told "sending soon", learns it
 * didn't go through and why; it's never retried and only leaves the queue
 * when dismissed.
 */
export type JobFailure = { message: string; code?: string };

/** Code for a job that kept failing while online — offered for a manual retry. */
export const GAVE_UP = 'gave_up';

// Online attempts before a job that keeps failing (an outage, or an error no
// retry can fix, like an unreadable photo) stops retrying on its own. Runs
// happen only on launch, reconnect and foreground, so this spans a while.
export const MAX_ATTEMPTS = 10;

export type QueueJob = (ReviewSubmissionJob | HelpfulVoteJob) & { failure?: JobFailure; attempts?: number };

let jobs: QueueJob[] | undefined;
let hydrating: Promise<QueueJob[]> | undefined;
const listeners = new Set<() => void>();
const EMPTY: QueueJob[] = [];

function notify(): void {
  listeners.forEach((listener) => listener());
}

function generateJobId(): string {
  return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`;
}

// Memoized: two cold-start callers (the processor on mount and an enqueue)
// must share one read, or the second parse overwrites the first caller's
// freshly pushed job and the next persist() makes that loss durable.
function hydrate(): Promise<QueueJob[]> {
  if (jobs) return Promise.resolve(jobs);
  hydrating ??= (async () => {
    const raw = await AsyncStorage.getItem(STORAGE_KEY);
    jobs = raw ? (JSON.parse(raw) as QueueJob[]) : [];
    // Notify here too, not just from persist(): a cold start with
    // already-persisted jobs (e.g. from a previous offline session) should
    // update a subscribed pending-count badge even before anything is
    // actually re-saved.
    notify();
    return jobs;
  })();
  return hydrating;
}

async function persist(): Promise<void> {
  // A fresh array on every save, so getQueueSnapshot() changes identity
  // whenever the queue (or a job inside it) changes — what
  // useSyncExternalStore compares.
  jobs = [...(jobs ?? [])];
  await AsyncStorage.setItem(STORAGE_KEY, JSON.stringify(jobs));
  notify();
}

/** For a UI badge/banner — call getQueueSnapshot() after hydrate() has run once (e.g. via useOfflineQueue). */
export function subscribeQueue(listener: () => void): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function getQueueSnapshot(): QueueJob[] {
  return jobs ?? EMPTY;
}

// Enqueue is only ever called from an authenticated screen (the review
// wizard, a helpful-vote tap), so a missing user id here means something is
// badly wrong with the calling code, not a normal runtime condition —
// failing loud is correct rather than silently tagging the job with nothing
// and letting it match whoever happens to be signed in later.
async function requireCurrentUserId(): Promise<string> {
  const userId = await getCurrentUserId();
  if (!userId) throw new Error('Cannot queue an offline job with no signed-in user');
  return userId;
}

export async function enqueueReviewSubmission(
  input: Pick<ReviewSubmissionJob, 'targetId' | 'idempotencyKey' | 'form'> & { reviewId?: string; pendingPhotoUris?: string[] }
): Promise<void> {
  const [, userId] = await Promise.all([hydrate(), requireCurrentUserId()]);
  // Always rebuild from the live `jobs`, never the array hydrate() returned:
  // a concurrent enqueue may have replaced it, and writing to the stale one
  // silently drops that other job.
  jobs = [
    ...(jobs ?? []),
    {
      type: 'review-submission',
    id: generateJobId(),
    userId,
    targetId: input.targetId,
    idempotencyKey: input.idempotencyKey,
    form: input.form,
    reviewId: input.reviewId,
      pendingPhotoUris: input.pendingPhotoUris ?? input.form.photos.map((photo) => photo.uri),
    },
  ];
  await persist();
}

export async function enqueueHelpfulVote(reviewId: string, voted: boolean): Promise<void> {
  const [, userId] = await Promise.all([hydrate(), requireCurrentUserId()]);
  // Last-write-wins: a newer toggle for the same review supersedes an
  // unsent older one rather than replaying both in order. Scoped to this
  // user's own prior jobs only — a different account's queued vote for the
  // same review (unlikely, but possible on a shared device) is untouched.
  jobs = [
    ...(jobs ?? []).filter((job) => !(job.type === 'helpful-vote' && job.reviewId === reviewId && job.userId === userId)),
    { type: 'helpful-vote', id: generateJobId(), userId, reviewId, voted },
  ];
  await persist();
}

/** Puts a job that gave up back in line and tries again now. */
export async function retryJob(id: string): Promise<void> {
  await hydrate();
  jobs = (jobs ?? []).map((j) => (j.id === id ? { ...j, failure: undefined, attempts: 0 } : j));
  await persist();
  await processQueue();
}

/** Removes a failed job the user has seen, cleaning up any photos it still held. */
export async function dismissJob(id: string): Promise<void> {
  await hydrate();
  const job = jobs?.find((j) => j.id === id);
  if (job?.type === 'review-submission') job.pendingPhotoUris.forEach(deletePersistedPhoto);
  jobs = (jobs ?? []).filter((j) => j.id !== id);
  await persist();
}

let processing = false;

/**
 * Drains everything it can *for the currently signed-in account*; jobs
 * belonging to a different account (queued by an earlier session on this
 * device, or awaiting the account that's still on this device) are left
 * queued untouched — replaying user A's job under user B's now-current
 * bearer token would publish it under the wrong account.
 */
export async function processQueue(): Promise<void> {
  if (processing) return;
  processing = true;
  try {
    const [queue, currentUserId, online] = await Promise.all([hydrate(), getCurrentUserId(), isConnected()]);
    // Offline, every job would fail the same way; running anyway would only
    // burn attempts that are meant to count real failures.
    if (!currentUserId || !online) return;
    for (const job of [...queue]) {
      if (job.userId !== currentUserId || job.failure) continue;
      const outcome = await runJob(job);
      if (outcome === 'done' && jobs) {
        jobs = jobs.filter((j) => j.id !== job.id);
        await persist();
      } else if (outcome === 'failed') {
        await persist();
      } else if (outcome === 'retry') {
        job.attempts = (job.attempts ?? 0) + 1;
        if (job.attempts >= MAX_ATTEMPTS) job.failure = { message: '', code: GAVE_UP };
        await persist();
      } else if (outcome === 'stop') {
        // The session itself is failing — every later job would 401 too.
        break;
      }
    }
  } finally {
    processing = false;
  }
}

type JobOutcome = 'done' | 'retry' | 'failed' | 'stop';

async function runJob(job: QueueJob): Promise<JobOutcome> {
  try {
    if (job.type === 'helpful-vote') {
      unwrap(
        await (job.voted
          ? apiClient.PUT('/api/v1/reviews/{id}/helpful', { params: { path: { id: job.reviewId } } })
          : apiClient.DELETE('/api/v1/reviews/{id}/helpful', { params: { path: { id: job.reviewId } } }))
      );
      return 'done';
    }

    if (!job.reviewId) {
      job.reviewId = await createReview(job.targetId, job.form, job.idempotencyKey);
      await persist();
    }
    job.pendingPhotoUris = await uploadPhotos(job.reviewId, job.pendingPhotoUris);
    await persist();
    invalidateReviewCaches(job.targetId, job.reviewId);
    return job.pendingPhotoUris.length === 0 ? 'done' : 'retry';
  } catch (err) {
    if (err instanceof ApiError && err.status === 401) return 'stop';
    if (isTransientFailure(err)) return 'retry';
    const apiError = err as ApiError;
    job.failure = { message: apiError.message, code: apiError.code };
    return 'failed';
  }
}
