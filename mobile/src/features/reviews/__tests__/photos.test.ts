import { downscaleFor } from '../photos';

jest.mock('expo-file-system', () => ({}));
jest.mock('expo-image-manipulator', () => ({}));
jest.mock('expo-image-picker', () => ({}));

it.each([
  [4032, 3024, { width: 1600 }], // landscape
  [3024, 4032, { height: 1600 }], // portrait: was left at 1600x2133
  [1080, 2400, { height: 1600 }], // tall screenshot: was not resized at all
  [1200, 900, undefined], // already small: never upscale
])('caps %ix%i by its longest edge', (width, height, expected) => {
  expect(downscaleFor(width, height)).toEqual(expected);
});
