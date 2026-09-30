const fs = require('fs');

// Push needs google-services.json on Android, but it identifies the Firebase
// project and stays out of git. On EAS it arrives as the GOOGLE_SERVICES_JSON
// file environment variable; locally, drop the file next to this config.
// Without either, the build still works — push registration just fails
// (and is logged) at runtime.
const LOCAL_GOOGLE_SERVICES = './google-services.json';

// Release builds (EAS preview/production) ship only the ARM ABIs real
// phones use, for the ≤25 MB APK budget (docs/mobile-plan.md §7). Local and
// development builds keep every ABI so x86_64 emulators still run them.
const RELEASE_PROFILES = new Set(['preview', 'production']);
const RELEASE_ARCHS = ['arm64-v8a', 'armeabi-v7a'];

function withReleaseArchs(plugins = []) {
  if (!RELEASE_PROFILES.has(process.env.EAS_BUILD_PROFILE)) return plugins;
  return plugins.map((plugin) =>
    Array.isArray(plugin) && plugin[0] === 'expo-build-properties'
      ? [plugin[0], { ...plugin[1], android: { ...plugin[1]?.android, buildArchs: RELEASE_ARCHS } }]
      : plugin
  );
}

module.exports = ({ config }) => {
  const googleServicesFile =
    process.env.GOOGLE_SERVICES_JSON ?? (fs.existsSync(LOCAL_GOOGLE_SERVICES) ? LOCAL_GOOGLE_SERVICES : undefined);
  return {
    ...config,
    android: { ...config.android, ...(googleServicesFile ? { googleServicesFile } : {}) },
    plugins: withReleaseArchs(config.plugins),
  };
};
