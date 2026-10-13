---
name: pr-writing
description: Draft or revise pull request titles and descriptions for qorv.in using the repository's standard title and description pattern. Use when a PR title, description, or both are requested.
---

# PR titles and descriptions

Base the content on the final changes in the requested branch or diff. Use a concise Conventional Commit title in the form `<type>: <imperative summary>`, choosing the type to match the actual change.

For a description, use exactly this structure:

```markdown
### Description

This PR aims to:

- Describe a concrete change starting with an active verb.
- Describe another relevant change.
```

When both are requested, put the title on the first line, followed by a blank line and the description. When only one is requested, return only that part.

Use as many bullets as the change warrants. Describe final behavior and relevant API, schema, documentation, or testing changes. Use Markdown links for referenced packages and bold formatting for API names. Keep each bullet on one physical line; do not manually wrap prose.

Do not invent validation results or add extra sections unless the user asks. Mention validation in a bullet when it helps assess the change, using checks actually completed. Drafting a title or description does not authorize creating, updating, or publishing a PR.
