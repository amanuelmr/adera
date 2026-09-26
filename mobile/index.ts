// Custom entry point instead of pointing package.json's "main" directly at
// expo-router/entry: the background push task must be defined at module
// scope in a file required early by the app, before expo-router's own
// bootstrap — see src/features/push/background-task.ts and
// https://docs.expo.dev/versions/latest/sdk/notifications/#inbackground.
import './src/features/push/background-task';
import 'expo-router/entry';
