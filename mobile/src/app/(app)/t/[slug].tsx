import { Redirect, useLocalSearchParams } from 'expo-router';

/** https://…/t/{slug} opened via App Links: the same place, in the native screen. */
export default function WebTargetLink() {
  const { slug } = useLocalSearchParams<{ slug: string }>();
  return <Redirect href={`/target/${slug}`} />;
}
