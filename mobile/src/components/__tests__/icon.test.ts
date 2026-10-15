import { Icon } from '../icon';

// Every icon in the registry is imported by path; a renamed or missing one
// fails here rather than at runtime.
it('loads every registered icon', () => {
  expect(Icon).toBeInstanceOf(Function);
});
