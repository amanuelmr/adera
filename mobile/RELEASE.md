# Releasing the Android app

Builds and store submission run on EAS (`eas.json`). Run the CLI as
`npx eas-cli@latest <command>`. Everything below is a one-time setup unless
marked per release.

## Build profiles

| Profile | Output | Use |
|---|---|---|
| `development` | APK with the dev client | Local testing with `npx expo start --dev-client`. Needed for push: Expo Go can't use this app's own Firebase config. |
| `preview` | APK, ARM only | Side-load to testers or share over Telegram. |
| `production` | AAB | Google Play. EAS increments `versionCode` on every build (`appVersionSource: remote`). |

Both release profiles point the app at `https://adera.amanuel.work`
(`EXPO_PUBLIC_API_BASE_URL`, `EXPO_PUBLIC_WEB_BASE_URL`). Without that the app
would call the emulator-only `10.0.2.2`.

Release builds turn on R8 minification and resource shrinking
(`expo-build-properties`). `app.config.js` limits `preview`/`production` to
`arm64-v8a` and `armeabi-v7a`, for the ≤ 25 MB APK budget
(docs/mobile-plan.md §7); Play delivers per-ABI splits of the AAB itself.

## One-time setup

1. **Project:** `eas init`, then `eas build:configure` if prompted. The
   Android package is `work.amanuel.adera`. It's permanent once it's on Play.
2. **Push (Firebase):** create an Android app for `work.amanuel.adera` in
   Firebase, download `google-services.json`, and store it as an EAS *file*
   environment variable named `GOOGLE_SERVICES_JSON` (visibility: secret)
   for the development, preview and production environments. Don't commit it.
   On the backend, upload the project's service-account key as a Render
   secret file and set `FCM_CREDENTIALS_FILE` (see `../DEPLOY.md`).
3. **Signing and App Links:** run the first `production` build and let EAS
   generate the keystore. Then `eas credentials -p android` → the profile →
   copy **SHA256 Fingerprint** and set it as `ANDROID_CERT_SHA256` on Render.
   If Play App Signing re-signs the app, also add the *app signing key*
   fingerprint from Play Console → Setup → App integrity (comma-separate
   both). Check it with
   `curl https://adera.amanuel.work/.well-known/assetlinks.json`.
4. **Play submission:** create a Google Play service account with release
   permissions and save its key as `mobile/play-service-account.json`
   (gitignored). The first upload of a new app must be done by hand in Play
   Console; `eas submit` works from the second one on.

## Per release

1. Bump `expo.version` in `app.json` (the user-visible version, e.g. `1.1.0`).
   `versionCode` is handled by EAS.
2. `npx eas-cli@latest build -p android --profile production`
3. **Internal testing:** `npx eas-cli@latest submit -p android --profile internal`.
   Test on real devices, especially Amharic rendering and push.
4. **Closed testing** (optional): promote the internal release in Play Console.
5. **Staged production:** `npx eas-cli@latest submit -p android --profile production`
   starts a rollout to **10%** of users (`releaseStatus: inProgress`,
   `rollout: 0.1`). Watch Play Console → Android vitals for crashes and ANRs,
   then raise it in Play Console (for example 25% → 50% → 100%) or halt it.
6. **Version gate:** once enough users are on the new release, retire broken
   old versions with `APP_ANDROID_MIN_VERSION` on Render (plain semver, e.g.
   `1.1.0`). Set `APP_ANDROID_LATEST_VERSION` and `APP_ANDROID_STORE_URL` to
   the Play listing. The store URL also turns on the "Get the app" banner on
   the web pages.

## Before the first public release

- Native-speaker review of the Amharic strings: `src/lib/locales/am.json`
  and `../internal/pages/i18n/am.json`. They were drafted without one.
- Check the APK/AAB size from the EAS build page against the 25 MB budget.
- Run Lighthouse on `https://adera.amanuel.work/t/<a real slug>` on a
  throttled mobile profile (docs/frontend-handoff.md §5).
