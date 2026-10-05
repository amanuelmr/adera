import { router } from 'expo-router';
import { useTranslation } from 'react-i18next';
import { StyleSheet, View } from 'react-native';

import { useRequireSignIn } from '@/auth/use-require-sign-in';
import { Button } from '@/components/button';
import { ThemedText } from '@/components/themed-text';
import { Spacing } from '@/constants/theme';

/** For empty results: turns "nothing here" into "add it". */
export function AddPlaceLink({ name, categoryId }: { name?: string; categoryId?: string }) {
  const { t } = useTranslation();
  const requireSignIn = useRequireSignIn();
  return (
    <View style={styles.container}>
      <ThemedText type="small" themeColor="textSecondary">
        {t('places.cantFind')}
      </ThemedText>
      <Button
        title={t('places.add')}
        variant="secondary"
        onPress={() =>
          requireSignIn(
            () => router.push({ pathname: '/add-place', params: { name: name ?? '', category: categoryId ?? '' } }),
            'addPlace'
          )
        }
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    paddingHorizontal: Spacing.four,
    gap: Spacing.two,
    alignItems: 'flex-start',
  },
});
