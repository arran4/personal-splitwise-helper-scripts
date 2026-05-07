# Agent Instructions

- Use minimal external dependencies.
- Stick to standard library whenever possible.
- Use `golang.org/x/...` for standard extensions.
- When creating tools, place them in `cmd/<tool>`.
- Use `.cache/` directory to store temporary or pulled API data.
- When retrieving from cache, always output the last modified date of the cache file.