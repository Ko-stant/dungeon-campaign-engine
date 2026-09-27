## Development Notes

- Server runs on port 8080 by default (configurable via `APP_PORT` env var)
- Static assets served from `internal/web/static/`
- Hot reloading configured via Air (`.air.toml`)
- All Go tools installed to `./.bin/` directory
- Uses Go modules with Go 1.27+ (templ is pinned as a go.mod tool: `go tool templ`)
- Never use emojis in code, comments, or console logs. They are only appropriate for html visuals.
- Follow Go best practices for error handling, logging, and concurrency
- When using locks/mutexes, always double check to make sure the flow properly locks and unlocks. Pay attention to edge cases where something can remain locked.
- When generating a compiled binary, never name the output "server" always use "dungeon-campaign-engine", or similar to avoid confusion with the cmd/server directory and the VSCode launch configuration named "Server". This is to avoid gitignore issues and  accidentally killing vscode processes
- Always run the application locally using `make dev` or some variant of it with additonal vars, e.g. `USE_LOBBY=true make dev`
- Always generate tests for new features as we go. Regression testing is important
- Client-side (TypeScript) work is test-first: write the failing bun:test before the code
- Clean up outdated docs and make sure to adjust asset_schemas when data structures change
- Prefer functionally separating new code into individual files where appropriate over making singular large files
- Never manually update *_templ.go files, only change the .templ file equivalents
- Always generate md files for TODO lists for tracking. They do not have to be perfect, but are required to have goals that can be tracked in case of power failure so we can return easily to them. After achieving a goal it is important to document how it was completed for posterity.
- Add/Update a timestamp in each MD file when making creating or making changes to them at the top of the file. This should include the time in addition to the date.