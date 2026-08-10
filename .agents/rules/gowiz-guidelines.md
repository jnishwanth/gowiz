# gowiz Rules & Code Quality Guidelines

## Code Style & Go Conventions
- Follow standard Go project structure (`internal/` for private packages).
- Format code with `gofmt` before committing.
- Keep functions short and single-purpose.
- Wrap errors with contextual information using `fmt.Errorf("...: %w", err)`.

## WiZ Protocol Safety
- Never hardcode dynamic IP addresses outside `wiz.FallbackIP`.
- Ensure all network calls support `context.Context` timeout cancellation.
- Clamping limits:
  - Brightness: 10% to 100%
  - Color Temp: 2200K to 6500K
  - RGB: 0 to 255
  - Scene ID: 1 to 32

## Testing Standards
- Unit test files must accompany every new package (`*_test.go`).
- Use `wiz.NewMockClient()` for TUI and client unit tests to avoid hardware dependencies.
- Ensure test coverage remains high across `internal/wiz` and `internal/tui`.
