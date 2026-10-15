/* eslint-disable import/first -- expo-font is mocked before the module loads */
jest.mock('expo-font', () => ({ useFonts: () => [true, null] }));

import { fontFamilyFor } from '../fonts';

it('maps each weight to the font file that carries it', () => {
  expect(fontFamilyFor(400, false)).toBe('Inter_400Regular');
  expect(fontFamilyFor(500, false)).toBe('Inter_400Regular');
  expect(fontFamilyFor(600, false)).toBe('Inter_600SemiBold');
  expect(fontFamilyFor(700, false)).toBe('Inter_700Bold');
});

it('uses Ethiopic regular and bold for Amharic', () => {
  expect(fontFamilyFor(400, true)).toBe('NotoSansEthiopic_400Regular');
  expect(fontFamilyFor(600, true)).toBe('NotoSansEthiopic_700Bold');
  expect(fontFamilyFor(700, true)).toBe('NotoSansEthiopic_700Bold');
});
