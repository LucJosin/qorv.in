# Project Instructions

- Use idiomatic Go.
- Follow existing package conventions.
- Avoid unnecessary dependencies.
- Run make test and make lint after changes.
- Prefer table-driven tests.
- Do not modify generated files.

# Pull request titles and descriptions

Whenever a PR title or description is requested, load and follow the local [pr-writing skill](.agents/skills/pr-writing/SKILL.md). Always use its pattern: a Conventional Commit title, followed by `### Description`, `This PR aims to:`, and a bullet list of concrete changes. Return only the requested parts and keep documentation prose on single physical lines.

# Code style recommendations

Split packages into focused files by responsibility, such as request validation, query parsing, cursor encoding, and page results. Avoid accumulating an entire feature in one large file.

Define reusable struct types at package scope instead of declaring anonymous response structs inside functions. Assign struct literals to variables before passing them to functions. For table-driven tests, keep the test-case declaration in the named `tests` variable as described below.

Avoid using `reflect`. Always ask for approval before introducing reflection, including new uses in code that already imports `reflect`.

Separate function calls and assignments from validation conditions: assign the result to a named variable before checking it, rather than using an initializer in the `if` statement or calling validation and error-classification functions directly in the condition. Keep conditions readable and avoid combining result extraction with multiple checks. Inline `cmp.Diff(want, got)` checks in tests remain preferred.

Name test functions using `Test<Function>_<Description>`, with an underscore separating the function or behavior under test from the scenario description, such as `TestCollectionList_PublicIDs`.

Use `github.com/google/go-cmp/cmp` for expected-versus-actual value comparisons in tests. Prefer `cmp.Diff(want, got)` and report differences with `(-want +got)`. Keep ordinary error checks and presence checks where a value comparison is not appropriate. Preserve the distinction between nil and empty slices unless the behavior under test treats them as equivalent.

Assign expected and actual values to named variables before comparing them. Reuse `want` and `got` when their types match. Use descriptive names such as `wantStatus`, `gotStatus`, `wantFields`, and `gotFields` when comparisons need different types. Do not introduce standalone blocks solely to scope comparison variables.

Place test helper types and functions after all test functions. Separate setup, execution, and assertion sections with blank lines.

Keep separate behaviors in separate test functions within the existing test file, and add comments only when the setup or intent is not clear from the code.

Use an external test package named `<package>_test` for test files whenever possible, such as `package v1_test`. Use the package under test only when access to unexported implementation details is necessary.

Place named (aliased) imports in a separate group at the bottom of the import block, separated from the other imports by a blank line, as in `collections_test.go`.

For table-driven Go tests, declare the test cases in a variable named `tests`, then iterate with `for _, tt := range tests`. Do not declare the test-case slice inline in the range expression. Write each field declaration and each populated field in test-case struct literals on its own line, including nested struct literals.

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

for _, tt := range tests {
	t.Run(tt.name, func(t *testing.T) {
		// Test the case.
	})
}
```
