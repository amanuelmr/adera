import AsyncStorage from '@react-native-async-storage/async-storage';
import * as Localization from 'expo-localization';
import i18next from 'i18next';
import { useEffect, useState } from 'react';
import { initReactI18next } from 'react-i18next';

import am from './locales/am.json';
import en from './locales/en.json';

// Scoped to the two languages this pass actually translated (trust-critical
// strings only — see the commit that introduced this file). Om/ti/so exist
// as review content-language options (components.schemas.Language) but
// aren't interface languages yet.
export const SUPPORTED_LANGUAGES = ['en', 'am'] as const;
export type AppLanguage = (typeof SUPPORTED_LANGUAGES)[number];

const STORAGE_KEY = 'adera.locale';

function isSupported(code: string | null | undefined): code is AppLanguage {
  return !!code && (SUPPORTED_LANGUAGES as readonly string[]).includes(code);
}

async function detectInitialLanguage(): Promise<AppLanguage> {
  const stored = await AsyncStorage.getItem(STORAGE_KEY);
  if (isSupported(stored)) return stored;

  const deviceLanguage = Localization.getLocales()[0]?.languageCode;
  return isSupported(deviceLanguage) ? deviceLanguage : 'en';
}

export async function setAppLanguage(language: AppLanguage): Promise<void> {
  await AsyncStorage.setItem(STORAGE_KEY, language);
  // eslint-disable-next-line import/no-named-as-default-member -- i18next's default export is the singleton instance; this is its documented usage.
  await i18next.changeLanguage(language);
}

let initialized: Promise<unknown> | undefined;

/** Idempotent — safe to call from more than one place (e.g. app start and tests). */
export function initI18n(): Promise<unknown> {
  if (!initialized) {
    initialized = detectInitialLanguage().then((lng) =>
      // eslint-disable-next-line import/no-named-as-default-member -- see above
      i18next.use(initReactI18next).init({
        resources: { en: { translation: en }, am: { translation: am } },
        lng,
        fallbackLng: 'en',
        interpolation: { escapeValue: false },
      })
    );
  }
  return initialized;
}

export function useI18nReady(): boolean {
  const [ready, setReady] = useState(false);
  useEffect(() => {
    let cancelled = false;
    initI18n().then(() => {
      if (!cancelled) setReady(true);
    });
    return () => {
      cancelled = true;
    };
  }, []);
  return ready;
}

export default i18next;
