import createClient from 'openapi-fetch';

import { apiBaseUrl } from './env';
import type { components, paths } from './schema';

export type ErrorEnvelope = components['schemas']['ErrorEnvelope'];
export type ApiErrorCode = NonNullable<ErrorEnvelope['error']>['code'];

/** Thrown by {@link unwrap} so call sites (e.g. TanStack Query) get a normal thrown error. */
export class ApiError extends Error {
  readonly status: number;
  readonly code: ApiErrorCode;
  readonly details?: Record<string, string>;
  readonly requestId?: string;

  constructor(status: number, envelope: ErrorEnvelope) {
    super(envelope.error?.message ?? `Request failed with status ${status}`);
    this.name = 'ApiError';
    this.status = status;
    this.code = envelope.error?.code ?? 'internal_error';
    this.details = envelope.error?.details;
    this.requestId = envelope.error?.request_id;
  }
}

export const apiClient = createClient<paths>({ baseUrl: apiBaseUrl });

type Envelope = { data?: unknown; meta?: components['schemas']['PageMeta'] };

/**
 * Unwraps an openapi-fetch `{data, error}` result, throwing {@link ApiError}
 * on failure. Every success response on this API is itself an envelope
 * (`{data, meta?}`, `meta.next_cursor` for cursor pagination) — this peels
 * that off too, so callers get the resource directly plus `meta` alongside it
 * rather than needing `.data.data`. Use inside TanStack Query query/mutation
 * functions, which expect a thrown error rather than a returned one.
 */
export function unwrap<T extends Envelope>(result: {
  data?: T;
  error?: ErrorEnvelope;
  response: Response;
}): { data: NonNullable<T['data']>; meta: T['meta'] } {
  if (result.error) {
    throw new ApiError(result.response.status, result.error);
  }
  if (result.data?.data === undefined) {
    throw new Error('Expected response data but received none');
  }
  return { data: result.data.data as NonNullable<T['data']>, meta: result.data.meta };
}
