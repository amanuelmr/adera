import AsyncStorage from '@react-native-async-storage/async-storage';
import { useEffect, useSyncExternalStore } from 'react';

export type Settings = {
  /** Text-first review lists; photos load only on request (docs/mobile-plan.md §7). */
  dataSaver: boolean;
};

const STORAGE_KEY = 'adera.settings.v1';
const DEFAULTS: Settings = { dataSaver: false };

let current: Settings = DEFAULTS;
let loading: Promise<void> | undefined;
const listeners = new Set<() => void>();

function notify(): void {
  listeners.forEach((listener) => listener());
}

export function loadSettings(): Promise<void> {
  loading ??= AsyncStorage.getItem(STORAGE_KEY)
    .then((raw) => {
      if (raw) current = { ...DEFAULTS, ...(JSON.parse(raw) as Partial<Settings>) };
      notify();
    })
    // Unreadable settings just mean defaults — never block the app on them.
    .catch(() => undefined);
  return loading;
}

export function getSettings(): Settings {
  return current;
}

export async function updateSettings(patch: Partial<Settings>): Promise<void> {
  await loadSettings();
  current = { ...current, ...patch };
  notify();
  await AsyncStorage.setItem(STORAGE_KEY, JSON.stringify(current));
}

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function useSettings(): Settings {
  useEffect(() => {
    loadSettings();
  }, []);
  return useSyncExternalStore(subscribe, getSettings);
}
