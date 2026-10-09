# Login investigation — October 9, 2026

Status: Code 366 is explained; authentication now uses one browser workflow
with headless terminal prompts or a visible window. Real-account password and
MFA success remain unverified against live X.

## Current account-login failure

An instrumented run of the actual CLI with the supplied username reached
`POST /onboarding/web/actions/begin_login` with both `username_or_email` and a
browser-generated `$castle_token`. X returned HTTP 200 with an application
error, and its visible form displayed:

> We’ve temporarily limited your login. Please try again later.

The page never advanced to password entry. This is evidence of a temporary
login limit, not evidence that headless login requires a visible browser.
The response did not identify the scope of the limit or when it expires.
No further account submissions were made after identifying it.

The CLI previously collapsed this text into `AUTHENTICATION_FAILED` with a
headed-login suggestion. It now reports `AUTH_RATE_LIMITED` and preserves
the instruction to try later. Generic page failures use
`AUTH_LOGIN_UNAVAILABLE`; hidden error text is ignored. Local Chromium
regressions exercise these rendered error states without contacting X.

The earlier phone attempt also used the wrong entry path: the current
`username_or_email` field is labelled “Email or username,” while phone login
has a separate “Continue with phone” control. The terminal prompt now matches
that field. The separate phone-login route is not implemented by this driver.
Unknown form timeouts indicate a limitation in form recognition, not proof of
an X requirement to use a visible browser.

The former experimental `auth login-api` implementation called the retired
classic onboarding flow. Updating its endpoint alone could not implement the
replacement: Jetfuel uses URL-encoded actions, binary responses, changing form
fields, and Castle device tokens. The implementation and public command have
been removed in favor of X's own web login.

## Final implementation decision

`twt auth login` defaults to a visible Chromium window for manual login.
`--headed` selects that default explicitly; `--headless` uses terminal prompts
for identity, password, and supported verification codes. Both
modes share one managed browser instance and application profile through
authentication and operation-contract capture. The chosen mode never changes
automatically, and Chromium is not relaunched between login and capture.

This lets X's frontend generate device tokens and submit its current login
forms. The CLI does not maintain a parallel implementation of the private
login API. Passwords and verification codes are hidden during terminal entry,
are not accepted in flags or environment variables, and are never saved.
Unsupported challenges, including CAPTCHA and security keys, fail with
instructions to rerun `twt auth login --headed`.

Contract refresh uses the existing profile headlessly. When authentication has
expired, it directs the user to log in explicitly rather than opening a
visible browser. After capture, the command saves authentication and candidate
contracts before validating the read operations. Successful validation promotes
the candidate contracts to active contracts. Validation failure preserves the
previous active contracts; the newly captured authentication is already saved.

## Historical reproduction

Before removal of the legacy implementation,
`go run ./.scratch/.sandbox/authprobe` called it without supplying account
credentials. The probe exited successfully only if the flow reached a
terminal input prompt. Repeated runs returned `AUTH_API_UNAVAILABLE` with X
code 366 before any prompt. The helper at
`.scratch/.sandbox/authprobe/main.go` was removed together with the legacy
`authapi` implementation. The command above records the historical experiment
and no longer runs in the current checkout.

## Live observations

- Direct HTTP `GET https://jf.x.com/onboarding/web?mode=login` returned HTTP 200.
  The binary response exposed `username_or_email` and `begin_login`.
- Playwright with Chromium 156 loaded
  `https://x.com/i/jf/onboarding/web?mode=login`. X rejected the default
  `HeadlessChrome` user agent with HTTP 403; the corresponding standard Chrome
  user agent loaded the page. A separate HTTP comparison reproduced this.
- One dummy identifier, `twt-login-probe@invalid.example`, was used throughout.
  No real account identifier, password, email code, or session credentials were
  provided. The dummy address uses the reserved `.example` domain.
- X's own frontend submitted
  `POST https://jf.x.com/onboarding/web/actions/begin_login` with form fields
  `username_or_email` and `$castle_token`. The generated token was about 5.9 KB;
  its value was neither printed nor saved.
- A controlled comparison intercepted a fresh browser submission and replayed
  it twice through the same HTTP client with the same headers and identifier.
  Removing only `$castle_token` returned HTTP 200 with an 82-byte binary
  “Something went wrong” response. Including the browser-generated token
  returned HTTP 200 and X's normal “Get the app to finish signing up using email”
  branch. HTTP success alone is therefore insufficient to validate login.

This establishes the active endpoint and the device-token dependency at the
identifier step. It does not establish successful password login or MFA for a
real account, nor prove that every headless-generated token is accepted.
A short headless token-generation step was considered during the investigation.
The final decision keeps the whole login in the same browser as contract
capture. A reliable browser-free token source was not established.

## Unified browser implementation probes

The final browser workflow was exercised through the CLI with both managed
Chromium and current Chromium, using isolated `XDG_CONFIG_HOME` and
`XDG_STATE_HOME` directories. These probes used only a dummy identifier in the
reserved `.example` domain. No real account password or verification code was
supplied.

The live identifier form contains an inert password input hidden by
`opacity: 0`. Login-field detection now filters that input so the CLI prompts
for the visible identifier. The current probe reaches that prompt and then
returns `AUTH_CHALLENGE_UNSUPPORTED`; it does not establish successful password
login or MFA. No specific post-identifier branch is claimed from this probe.

## Current primary-source corroboration

- [Twifork migration evidence, July 30](https://github.com/PawiX25/twifork/commit/3294c1ebf020761c95c877dc36ed06c24a944e88)
- [Mautrix Jetfuel route update, September 30](https://github.com/mautrix/twitter/pull/153)
- [Mautrix login implementation, October 6 revision](https://github.com/mautrix/twitter/blob/fea0bb26af1a5b7529588dee9ab4fa1f6e10be3b/pkg/twittermeow/login_jetfuel.go)
- [Mautrix's browser/webview token requirement](https://github.com/mautrix/twitter/blob/fea0bb26af1a5b7529588dee9ab4fa1f6e10be3b/pkg/twittermeow/castle_token.go)
- [Emusks login documentation and limitations](https://emusks.tiago.zip/jetfuel/login)

Mautrix's current implementation carries browser/webview-generated Castle tokens
for Jetfuel form submissions. Emusks also describes a browser token dependency;
its documentation and shipped source differ on routes, and its statements about
initial stages should not be read as proof of full browser-free account login.
