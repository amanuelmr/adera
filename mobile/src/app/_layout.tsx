import { PersistQueryClientProvider } from '@tanstack/react-query-persist-client';
import { DarkTheme, DefaultTheme, Stack, ThemeProvider } from 'expo-router';
import { useColorScheme } from 'react-native';

import { AuthProvider, useAuth } from '@/auth/context';
import { SplashScreenController } from '@/components/splash-screen-controller';
import { useVersionGate } from '@/features/app-version/queries';
import { UpdateRequiredScreen } from '@/features/app-version/update-required-screen';
import { useEthiopicFonts } from '@/lib/fonts';
import i18n, { useI18nReady } from '@/lib/i18n';
import { persistOptions, queryClient } from '@/lib/query-client';

export default function RootLayout() {
  const colorScheme = useColorScheme();
  return (
    <PersistQueryClientProvider client={queryClient} persistOptions={persistOptions}>
      <AuthProvider>
        <ThemeProvider value={colorScheme === 'dark' ? DarkTheme : DefaultTheme}>
          <AppGate />
        </ThemeProvider>
      </AuthProvider>
    </PersistQueryClientProvider>
  );
}

function AppGate() {
  const { status } = useAuth();
  const versionGate = useVersionGate();
  const i18nReady = useI18nReady();
  // Always called (Rules of Hooks) — these fonts load in the background
  // regardless of locale, but only block startup when they're actually
  // needed (i.e. Amharic is the resolved language), per
  // docs/frontend-handoff.md §7's "load only when locale = am".
  const ethiopicFontsLoaded = useEthiopicFonts();
  const needsEthiopicFonts = i18nReady && i18n.language === 'am';

  // A version-check failure (offline on first launch, etc.) shouldn't trap
  // the user on the splash screen forever — only an explicit
  // update_required=true blocks.
  const ready = status !== 'loading' && !versionGate.isPending && i18nReady && (!needsEthiopicFonts || ethiopicFontsLoaded);

  return (
    <>
      <SplashScreenController ready={ready} />
      {ready && versionGate.data?.update_required ? (
        <UpdateRequiredScreen storeUrl={versionGate.data.store_url} />
      ) : (
        <RootNavigator />
      )}
    </>
  );
}

function RootNavigator() {
  const { status } = useAuth();

  // Reading is never gated behind auth (docs/mobile-plan.md §6): (app) is
  // always reachable, and only its write screens are protected — see
  // (app)/_layout.tsx. Sign-in opens over whatever the reader was looking at.
  return (
    <Stack screenOptions={{ headerShown: false }}>
      <Stack.Screen name="(app)" />
      <Stack.Protected guard={status === 'signedOut'}>
        <Stack.Screen name="(auth)" options={{ presentation: 'modal' }} />
      </Stack.Protected>
    </Stack>
  );
}
