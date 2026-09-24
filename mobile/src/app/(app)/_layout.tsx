import { Stack } from 'expo-router';

export default function AppLayout() {
  return (
    <Stack screenOptions={{ headerShown: false }}>
      <Stack.Screen name="index" />
      <Stack.Screen name="verify" options={{ headerShown: true, title: 'Verify email' }} />
      <Stack.Screen name="sessions" options={{ headerShown: true, title: 'Your devices' }} />
    </Stack>
  );
}
