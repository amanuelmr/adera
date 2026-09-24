import { Stack } from 'expo-router';

export default function AppLayout() {
  return (
    <Stack>
      <Stack.Screen name="(tabs)" options={{ headerShown: false }} />
      <Stack.Screen name="search" options={{ title: 'Search' }} />
      <Stack.Screen name="target/[idOrSlug]" options={{ title: '' }} />
      <Stack.Screen name="verify" options={{ title: 'Verify email' }} />
      <Stack.Screen name="sessions" options={{ title: 'Your devices' }} />
    </Stack>
  );
}
