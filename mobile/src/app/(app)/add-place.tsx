import { router, useLocalSearchParams } from 'expo-router';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ScrollView, StyleSheet, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { ApiError } from '@/api/client';
import type { components } from '@/api/schema';
import { Button } from '@/components/button';
import { TextField } from '@/components/text-field';
import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import { Spacing } from '@/constants/theme';
import { useAreas, useCities, useSubmitPlace } from '@/features/places/queries';
import { FilterChip } from '@/features/search/filter-chip';
import { TARGET_TYPES } from '@/features/search/types';
import { categoryName } from '@/features/targets/category-name';
import { useCategories } from '@/features/targets/queries';
import { locateDevice } from '@/features/targets/use-device-location';
import i18n from '@/lib/i18n';

type TargetType = components['schemas']['TargetType'];

// Same bounds as the API (internal/targets/handler.go).
const NAME_MIN = 2;
const NAME_MAX = 160;

function localName(item: { name?: string; name_am?: string }): string {
  return (i18n.language === 'am' && item.name_am) || item.name || '';
}

export default function AddPlaceScreen() {
  const params = useLocalSearchParams<{ name?: string; category?: string }>();
  const { t } = useTranslation();
  const categories = useCategories();
  const cities = useCities();
  const submit = useSubmitPlace();

  const [name, setName] = useState(params.name ?? '');
  const [categoryId, setCategoryId] = useState<string | undefined>(params.category || undefined);
  const [type, setType] = useState<TargetType>('business');
  const [cityId, setCityId] = useState<string>();
  const [areaId, setAreaId] = useState<string>();
  const [landmark, setLandmark] = useState('');
  const [coords, setCoords] = useState<{ latitude: number; longitude: number }>();
  const [locating, setLocating] = useState(false);
  const [locationError, setLocationError] = useState(false);
  const areas = useAreas(cityId);

  const trimmed = name.trim();
  const canSubmit = trimmed.length >= NAME_MIN && trimmed.length <= NAME_MAX && !!categoryId;

  async function useMyLocation() {
    setLocating(true);
    setLocationError(false);
    const state = await locateDevice();
    setLocating(false);
    if (state.status === 'ready') setCoords({ latitude: state.latitude, longitude: state.longitude });
    else setLocationError(true);
  }

  function onSubmit() {
    if (!canSubmit) return;
    submit.mutate({
      name: trimmed,
      category_id: categoryId!,
      target_type: type,
      city_id: cityId,
      area_id: areaId,
      address_text: landmark.trim() || undefined,
      latitude: coords?.latitude,
      longitude: coords?.longitude,
    });
  }

  if (submit.isSuccess) {
    return (
      <ThemedView style={styles.centered}>
        <ThemedText type="subtitle">{t('places.submittedTitle')}</ThemedText>
        <ThemedText type="small" themeColor="textSecondary" style={styles.center}>
          {t('places.submittedBody')}
        </ThemedText>
        <Button title={t('common.back')} onPress={() => router.back()} />
      </ThemedView>
    );
  }

  const error = submit.error
    ? submit.error instanceof ApiError && submit.error.status === 429
      ? t('places.rateLimited')
      : submit.error instanceof ApiError && submit.error.status === 422
        ? t('errors.validation')
        : t('errors.generic')
    : undefined;

  return (
    <ThemedView style={styles.container}>
      <SafeAreaView style={styles.safeArea} edges={['bottom']}>
        <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
          <ThemedText type="small" themeColor="textSecondary">
            {t('places.intro')}
          </ThemedText>

          <TextField
            label={t('places.name')}
            value={name}
            onChangeText={setName}
            maxLength={NAME_MAX}
            autoCapitalize="words"
          />

          <ThemedText type="smallBold">{t('search.category')}</ThemedText>
          <View style={styles.chips}>
            {(categories.data ?? []).map((category) => (
              <FilterChip
                key={category.id}
                label={categoryName(category) ?? ''}
                selected={categoryId === category.id}
                onPress={() => setCategoryId(category.id)}
              />
            ))}
          </View>

          <ThemedText type="smallBold">{t('search.typeLabel')}</ThemedText>
          <View style={styles.chips}>
            {TARGET_TYPES.map((value) => (
              <FilterChip
                key={value}
                label={t(`search.type.${value}`)}
                selected={type === value}
                onPress={() => setType(value)}
              />
            ))}
          </View>

          <ThemedText type="smallBold">{t('places.city')}</ThemedText>
          <View style={styles.chips}>
            {(cities.data ?? []).map((city) => (
              <FilterChip
                key={city.id}
                label={localName(city)}
                selected={cityId === city.id}
                onPress={() => {
                  setCityId(city.id === cityId ? undefined : city.id);
                  setAreaId(undefined);
                }}
              />
            ))}
          </View>

          {cityId && (areas.data ?? []).length > 0 ? (
            <>
              <ThemedText type="smallBold">{t('places.area')}</ThemedText>
              <View style={styles.chips}>
                {(areas.data ?? []).map((area) => (
                  <FilterChip
                    key={area.id}
                    label={localName(area)}
                    selected={areaId === area.id}
                    onPress={() => setAreaId(area.id === areaId ? undefined : area.id)}
                  />
                ))}
              </View>
            </>
          ) : null}

          <TextField
            label={t('places.landmark')}
            placeholder={t('places.landmarkPlaceholder')}
            value={landmark}
            onChangeText={setLandmark}
            maxLength={200}
          />

          <Button
            title={coords ? t('places.locationSet') : t('places.useLocation')}
            variant="secondary"
            onPress={useMyLocation}
            loading={locating}
          />
          {locationError ? (
            <ThemedText type="small" themeColor="textSecondary">
              {t('nearMe.errorBody')}
            </ThemedText>
          ) : null}

          {error ? (
            <ThemedText type="small" style={styles.error}>
              {error}
            </ThemedText>
          ) : null}

          <Button title={t('places.submit')} onPress={onSubmit} disabled={!canSubmit} loading={submit.isPending} />
        </ScrollView>
      </SafeAreaView>
    </ThemedView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  safeArea: {
    flex: 1,
  },
  content: {
    padding: Spacing.four,
    gap: Spacing.three,
  },
  chips: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: Spacing.two,
  },
  centered: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    gap: Spacing.three,
    padding: Spacing.four,
  },
  center: {
    textAlign: 'center',
  },
  error: {
    color: '#D64545',
  },
});
