export type CachedQueryState = { data?: unknown; dataUpdatedAt: number; isRefetchError: boolean };

/**
 * Cached data must say it's cached (docs/mobile-plan.md §5: "an honest saved
 * N hours ago banner"): true when what's on screen came from the device cache
 * and couldn't be refreshed — offline, or the refresh failed.
 */
export function shouldShowCachedBanner(query: CachedQueryState, online: boolean): boolean {
  if (query.data == null || !query.dataUpdatedAt) return false;
  return !online || query.isRefetchError;
}
