// Codes the backend's OAuth callback can send via ?error= — shared by the
// full-page callback and the popup flow, which both need to turn one of
// these into a user-facing message.
const ERROR_MESSAGE_KEYS: Record<string, string> = {
  invalid_state: 'auth.errorAuthFailed',
  missing_verifier: 'auth.errorAuthFailed',
  token_exchange_failed: 'auth.errorConnectFailed',
  fetch_user_failed: 'auth.errorFetchUserFailed',
  create_user_failed: 'auth.errorCreateUserFailed',
  issue_token_failed: 'auth.errorIssueTokenFailed',
  // sso decides who may use this app (see app/membership on the API) — these
  // two are login rejections, not technical failures, so they get their own
  // wording rather than falling into errorUnexpected.
  no_account: 'auth.errorNoAccount',
  membership_inactive: 'auth.errorMembershipInactive',
}

export const authErrorMessageKey = (code?: string | null): string =>
  ERROR_MESSAGE_KEYS[code ?? ''] ?? 'auth.errorUnexpected'
