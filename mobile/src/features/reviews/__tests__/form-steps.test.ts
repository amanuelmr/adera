import { canProceedFromStep, dateChoiceFor, daysAgo, REVIEW_STEP_COUNT, writingPromptFor } from '../form-steps';
import { INITIAL_REVIEW_FORM, type ReviewFormState } from '../types';

const today = new Date(2026, 9, 1); // Oct 1, 2026, local
const criteriaOk = { pending: false, missingRequired: 0 };

function form(overrides: Partial<ReviewFormState>): ReviewFormState {
  return { ...INITIAL_REVIEW_FORM, ...overrides };
}

it('has four steps', () => {
  expect(REVIEW_STEP_COUNT).toBe(4);
});

it.each([
  [0, form({}), criteriaOk, false],
  [0, form({ overallRating: 4 }), criteriaOk, true],
  [1, form({}), { pending: true, missingRequired: 0 }, false],
  [1, form({}), { pending: false, missingRequired: 1 }, false],
  [1, form({}), criteriaOk, true],
  [2, form({ body: '   too short   ' }), criteriaOk, false],
  [2, form({ body: 'Long enough to tell people something.' }), criteriaOk, true],
  [3, form({}), criteriaOk, true],
  [3, form({ experienceDate: '2026-13-01' }), criteriaOk, false],
  [3, form({ pricePaid: 'abc' }), criteriaOk, false],
  [3, form({ incentiveType: 'other' }), criteriaOk, false],
  [3, form({ materialConnection: 'other', disclosureDetails: 'I know the owner' }), criteriaOk, true],
  [3, form({ incentiveType: 'discount' }), criteriaOk, true],
  [4, form({ overallRating: 5 }), criteriaOk, false],
] as const)('step %p with %o → %p', (step, value, criteria, expected) => {
  expect(canProceedFromStep(step, value, criteria)).toBe(expected);
});

it('formats relative days as local calendar dates, across month boundaries', () => {
  expect(daysAgo(0, today)).toBe('2026-10-01');
  expect(daysAgo(1, today)).toBe('2026-09-30');
  expect(daysAgo(1, new Date(2026, 0, 1))).toBe('2025-12-31');
});

it.each([
  ['', undefined],
  ['  ', undefined],
  ['2026-10-01', 'today'],
  ['2026-09-30', 'yesterday'],
  ['2026-09-20', 'pick'],
  ['not a date', 'pick'],
])('date %p → chip %p', (value, expected) => {
  expect(dateChoiceFor(value, today)).toBe(expected);
});

it.each([
  [undefined, undefined],
  [1, 'low'],
  [2, 'low'],
  [3, 'mid'],
  [4, 'high'],
  [5, 'high'],
])('rating %p → prompt %p', (rating, expected) => {
  expect(writingPromptFor(rating)).toBe(expected);
});
