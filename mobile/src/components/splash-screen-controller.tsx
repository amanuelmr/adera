import * as SplashScreen from 'expo-splash-screen';
import { useEffect } from 'react';

SplashScreen.preventAutoHideAsync();

/** Keeps the splash screen up until every startup check (auth, version gate, ...) is ready. */
export function SplashScreenController({ ready }: { ready: boolean }) {
  useEffect(() => {
    if (ready) {
      SplashScreen.hideAsync();
    }
  }, [ready]);

  return null;
}
