// The FCM data message from internal/notifications/fcm.go: flat string
// fields (event_type, subject_type, subject_id, …) plus `data`, the event's
// own fields as a JSON string. Unverified on a real device: the fields can
// arrive as direct keys or JSON-encoded under `dataString` depending on
// platform/OS version, so both are handled.
export type PushPayload = { eventType?: string; data: Record<string, string> };

export function parsePushPayload(payloadData: unknown): PushPayload {
  if (!payloadData || typeof payloadData !== 'object') return { data: {} };
  let record = payloadData as Record<string, unknown>;
  if (typeof record.event_type !== 'string' && typeof record.dataString === 'string') {
    try {
      record = JSON.parse(record.dataString);
    } catch {
      return { data: {} };
    }
  }
  return {
    eventType: typeof record.event_type === 'string' ? record.event_type : undefined,
    data: parseData(record.data),
  };
}

function parseData(raw: unknown): Record<string, string> {
  let value = raw;
  if (typeof value === 'string') {
    try {
      value = JSON.parse(value);
    } catch {
      return {};
    }
  }
  if (!value || typeof value !== 'object') return {};
  const out: Record<string, string> = {};
  for (const [key, v] of Object.entries(value)) if (typeof v === 'string') out[key] = v;
  return out;
}
