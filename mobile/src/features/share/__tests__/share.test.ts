import { buildReviewLink, buildReviewShareText, buildTargetLink, buildTargetShareText } from '../share';

jest.mock('@/lib/i18n', () => ({
  __esModule: true,
  default: {
    t: (key: string, params: Record<string, unknown> = {}) =>
      `${key}(${Object.entries(params)
        .map(([k, v]) => `${k}=${v}`)
        .join(',')})`,
  },
}));

it('links to the public web pages, which App Links open in the app', () => {
  expect(buildTargetLink('tomoca-piassa')).toBe('https://adera.amanuel.work/t/tomoca-piassa');
  expect(buildReviewLink('0b7e…')).toBe('https://adera.amanuel.work/r/0b7e%E2%80%A6');
});

it('only shares a rating the profile would show (3+ reviews)', () => {
  expect(buildTargetShareText({ name: 'Tomoca', average_rating: 5, review_count: 1 }, 'L')).toBe(
    'share.target(name=Tomoca)\nL'
  );
  expect(buildTargetShareText({ name: 'Tomoca', average_rating: 4.26, review_count: 12 }, 'L')).toBe(
    'share.targetRated(name=Tomoca,rating=4.3)\nL'
  );
});

it('trims long quotes with an ellipsis', () => {
  const text = buildReviewShareText({ body: 'x'.repeat(300), overall_rating: 4 }, 'Tomoca', 'L');
  expect(text).toContain(`quote=${'x'.repeat(140)}…`);
});
