// expo-env.d.ts is gitignored (Expo generates it) but only `expo start`'s dev
// server actually writes it — there's no one-shot CLI command for it, so a
// fresh checkout that only ever runs `tsc`/`eslint`/CI never gets one.
// Content is static (references the expo/types package, nothing project-
// specific), so this postinstall step just writes it directly rather than
// requiring everyone to run the dev server once first.
const fs = require('fs');
const path = require('path');

const target = path.join(__dirname, '..', 'expo-env.d.ts');
const contents = '/// <reference types="expo/types" />\n\n// NOTE: This file should not be edited and should be in your git ignore\n';

if (!fs.existsSync(target)) {
  fs.writeFileSync(target, contents);
}
