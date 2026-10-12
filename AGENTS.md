# Project Instructions

- Use idiomatic Go.
- Follow existing package conventions.
- Avoid unnecessary dependencies.
- Run make test and make lint after changes.
- Prefer table-driven tests.
- Do not modify generated files.

# Code style recommendations

Name test functions using `Test<Function>_<Description>`, with an underscore separating the function or behavior under test from the scenario description, such as `TestCollectionList_PublicIDs`.

Use `github.com/google/go-cmp/cmp` for expected-versus-actual value comparisons in tests. Prefer `cmp.Diff(want, got)` and report differences with `(-want +got)`. Keep ordinary error checks and presence checks where a value comparison is not appropriate. Preserve the distinction between nil and empty slices unless the behavior under test treats them as equivalent.

Use an external test package named `<package>_test` for test files whenever possible, such as `package v1_test`. Use the package under test only when access to unexported implementation details is necessary.

Place named (aliased) imports in a separate group at the bottom of the import block, separated from the other imports by a blank line, as in `collections_test.go`.

For table-driven Go tests, declare the test cases in a variable named `tests`, then iterate with `for _, test := range tests`. Do not declare the test-case slice inline in the range expression. Write each field declaration and each populated field in test-case struct literals on its own line, including nested struct literals.

```go
tests := []struct {
	name string
	want int
}{
	{
		name: "example",
		want: 1,
	},
}

for _, test := range tests {
	t.Run(test.name, func(t *testing.T) {
		// Test the case.
	})
}
```
