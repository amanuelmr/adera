import * as SplashScreen from 'expo-splash-screen';
import { useEffect } from 'react';

import { useAuth } from '@/auth/context';

SplashScreen.preventAutoHideAsync();

/** Keeps the splash screen up until auth status resolves from secure storage. */
export function SplashScreenController() {
  const { status } = useAuth();

  useEffect(() => {
    if (status !== 'loading') {
      SplashScreen.hideAsync();
    }
  }, [status]);

  return null;
}
