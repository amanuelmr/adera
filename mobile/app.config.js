const fs = require('fs');

// Push needs google-services.json on Android, but it identifies the Firebase
// project and stays out of git. On EAS it arrives as the GOOGLE_SERVICES_JSON
// file environment variable; locally, drop the file next to this config.
// Without either, the build still works — push registration just fails
// (and is logged) at runtime.
const LOCAL_GOOGLE_SERVICES = './google-services.json';

module.exports = ({ config }) => {
  const googleServicesFile =
    process.env.GOOGLE_SERVICES_JSON ?? (fs.existsSync(LOCAL_GOOGLE_SERVICES) ? LOCAL_GOOGLE_SERVICES : undefined);
  return {
    ...config,
    android: { ...config.android, ...(googleServicesFile ? { googleServicesFile } : {}) },
  };
};
