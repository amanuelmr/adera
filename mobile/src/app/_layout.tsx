import { QueryClientProvider } from '@tanstack/react-query';
import { DarkTheme, DefaultTheme, Stack, ThemeProvider } from 'expo-router';
import { useColorScheme } from 'react-native';

import { AuthProvider, useAuth } from '@/auth/context';
import { SplashScreenController } from '@/components/splash-screen-controller';
import { useVersionGate } from '@/features/app-version/queries';
import { UpdateRequiredScreen } from '@/features/app-version/update-required-screen';
import { queryClient } from '@/lib/query-client';

export default function RootLayout() {
  const colorScheme = useColorScheme();
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <ThemeProvider value={colorScheme === 'dark' ? DarkTheme : DefaultTheme}>
          <AppGate />
        </ThemeProvider>
      </AuthProvider>
    </QueryClientProvider>
  );
}

function AppGate() {
  const { status } = useAuth();
  const versionGate = useVersionGate();
  // A version-check failure (offline on first launch, etc.) shouldn't trap
  // the user on the splash screen forever — only an explicit
  // update_required=true blocks.
  const ready = status !== 'loading' && !versionGate.isPending;

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

  return (
    <Stack screenOptions={{ headerShown: false }}>
      <Stack.Protected guard={status === 'signedIn'}>
        <Stack.Screen name="(app)" />
      </Stack.Protected>
      <Stack.Protected guard={status === 'signedOut'}>
        <Stack.Screen name="(auth)" />
      </Stack.Protected>
    </Stack>
  );
}
