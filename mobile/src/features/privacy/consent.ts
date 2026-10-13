/** True when the account hasn't accepted the privacy notice the server currently serves. */
export function needsPrivacyConsent(notice: { accepted_version?: string; current_version: string }): boolean {
  return notice.accepted_version !== notice.current_version;
}
