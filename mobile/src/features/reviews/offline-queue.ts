import AsyncStorage from '@react-native-async-storage/async-storage';

import { apiClient, ApiError } from '@/api/client';
import { createReview, uploadPhotos } from './queries';
import type { ReviewFormState } from './types';

const STORAGE_KEY = 'adera.offline-queue.v1';

export type ReviewSubmissionJob = {
  type: 'review-submission';
  id: string;
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
  reviewId: string;
  voted: boolean;
};

export type QueueJob = ReviewSubmissionJob | HelpfulVoteJob;

let jobs: QueueJob[] | undefined;
const listeners = new Set<() => void>();

function generateJobId(): string {
  return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`;
}

async function hydrate(): Promise<QueueJob[]> {
  if (!jobs) {
    const raw = await AsyncStorage.getItem(STORAGE_KEY);
    jobs = raw ? (JSON.parse(raw) as QueueJob[]) : [];
    // Notify here too, not just from persist(): a cold start with
    // already-persisted jobs (e.g. from a previous offline session) should
    // update a subscribed pending-count badge even before anything is
    // actually re-saved.
    listeners.forEach((listener) => listener());
  }
  return jobs;
}

async function persist(): Promise<void> {
  await AsyncStorage.setItem(STORAGE_KEY, JSON.stringify(jobs ?? []));
  listeners.forEach((listener) => listener());
}

/** For a UI badge/banner — call getQueueSnapshot() after hydrate() has run once (e.g. via useOfflineQueue). */
export function subscribeQueue(listener: () => void): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function getQueueSnapshot(): QueueJob[] {
  return jobs ?? [];
}

export async function enqueueReviewSubmission(
  input: Pick<ReviewSubmissionJob, 'targetId' | 'idempotencyKey' | 'form'> & { reviewId?: string; pendingPhotoUris?: string[] }
): Promise<void> {
  const queue = await hydrate();
  queue.push({
    type: 'review-submission',
    id: generateJobId(),
    targetId: input.targetId,
    idempotencyKey: input.idempotencyKey,
    form: input.form,
    reviewId: input.reviewId,
    pendingPhotoUris: input.pendingPhotoUris ?? input.form.photos.map((photo) => photo.uri),
  });
  await persist();
}

export async function enqueueHelpfulVote(reviewId: string, voted: boolean): Promise<void> {
  const queue = await hydrate();
  // Last-write-wins: a newer toggle for the same review supersedes an
  // unsent older one rather than replaying both in order.
  jobs = queue.filter((job) => !(job.type === 'helpful-vote' && job.reviewId === reviewId));
  jobs.push({ type: 'helpful-vote', id: generateJobId(), reviewId, voted });
  await persist();
}

let processing = false;

/** Drains everything it can; jobs that still fail for network reasons stay queued. */
export async function processQueue(): Promise<void> {
  if (processing) return;
  processing = true;
  try {
    const queue = await hydrate();
    for (const job of [...queue]) {
      const resolved = await runJob(job);
      if (resolved && jobs) {
        jobs = jobs.filter((j) => j.id !== job.id);
        await persist();
      }
    }
  } finally {
    processing = false;
  }
}

/** Returns true when the job is done (succeeded, or failed for a reason retrying won't fix). */
async function runJob(job: QueueJob): Promise<boolean> {
  try {
    if (job.type === 'helpful-vote') {
      await (job.voted
        ? apiClient.PUT('/api/v1/reviews/{id}/helpful', { params: { path: { id: job.reviewId } } })
        : apiClient.DELETE('/api/v1/reviews/{id}/helpful', { params: { path: { id: job.reviewId } } }));
      return true;
    }

    if (!job.reviewId) {
      job.reviewId = await createReview(job.targetId, job.form, job.idempotencyKey);
      await persist();
    }
    job.pendingPhotoUris = await uploadPhotos(job.reviewId, job.pendingPhotoUris);
    await persist();
    return job.pendingPhotoUris.length === 0;
  } catch (err) {
    // A real response came back (even a rejection) — offline retry can't
    // change that outcome, so stop queuing it rather than looping forever.
    return err instanceof ApiError;
  }
}
