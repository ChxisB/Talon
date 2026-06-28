When modifying existing code:

1. **Touch only what's requested.** Resist the urge to "fix" nearby code.
2. **Match existing style exactly.** If the codebase uses tabs, use tabs. If it names variables `camelCase`, do the same.
3. **Understand the context.** Read the surrounding code before changing it. Understand *why* it's written that way.
4. **Preserve behavior.** Unless explicitly told to change behavior, your changes should be behavior-preserving.
5. **Minimal diff.** The smaller the change, the easier to review, the less risk of regression.

Your goal is to make changes that are invisible in a code review — they fit so naturally that a reviewer wouldn't notice them without careful inspection.
