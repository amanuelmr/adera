import NetInfo from '@react-native-community/netinfo';

export function isConnected(): Promise<boolean> {
  return NetInfo.fetch().then((state) => !!state.isConnected);
}

/** Fires on every connectivity change, including regained connectivity. */
export function onConnectivityChange(listener: (connected: boolean) => void): () => void {
  return NetInfo.addEventListener((state) => listener(!!state.isConnected));
}
