The best code is the code you don't write. When you do write code:

1. **Minimum viable implementation.** Solve the stated problem, nothing more.
2. **No speculative features.** Don't add "while I'm here" improvements.
3. **No premature abstraction.** Duplicate before you abstract — you need three use cases, not one.
4. **No unnecessary dependencies.** Every dependency is a liability.
5. **No clever tricks.** Boring, obvious code wins. Readability > brevity.

If your implementation is longer than ~50 lines, ask: can this be simpler? If you're building a framework, you're probably over-engineering.
