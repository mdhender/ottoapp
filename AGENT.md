# AGENT.md

## Commands
- Build: `go build`
- Run server: `./ottoapp serve --database /path/to/db --host localhost --port 29631`
- Run backend API server: `cd ottobe && go build && ./ottobe --dev`
- Build for Linux: `GOOS=linux GOARCH=amd64 go build -o ottoapp.exe`
- Tests: `go test ./...`
- Run single test: `go test -v ./path/to/package -run TestName`
- Format code: `go fmt ./...`
- Frontend dev server: `cd ottofe && npm run dev`

## Code Style
- Standard Go formatting using `gofmt`
- Imports organized by stdlib first, then external packages
- Error handling: return errors to caller, log.Fatal only in main
- Function comments use Go standard format `// FunctionName does X`
- Variable naming follows camelCase
- File structure follows standard Go package conventions
- Errors defined in domains/errors.go
- Authentication handled in domains/auth.go

## Project Structure
- assets/: Static files (CSS, JS, images)
- components/: UI components
- domains/: Core domain logic
- stores/: Data storage implementations
- bin/: Command line utilities
- ottofe/: The new React + Vite + Tailwind front end code
- ottobe/: The new Go RESTish back end code

## Front End
- We will never use SSR
- Node.js version 23.11.0 (specified in ottofe/.nvmrc)
- Run `cd ottofe && nvm use` to ensure correct Node version
- Using Tailwind CSS v4 with direct import (no tailwind.config.js)
- Path aliases configured in vite.config.js: '@' and '@components'

## Architecture Changes
- Transitioning from Go HTML templates to React + Vite + Tailwind frontend
- Frontend served by `npm run dev` in development, Nginx in production
- Backend updated to serve only API endpoints
- Frontend connects to https://localhost:29631/api/... in development
- In production, frontend connects to /api/... with Nginx proxy to backend

## Bash Scripts
- Always use `${VARIABLE}` with curly braces for all variables
- Always quote variable references: "${VARIABLE}"
- Use `set -e` for early exit on errors
- Include descriptive echo statements with emoji for visual feedback
- Test endpoints in sequence with explicit validation
- Exit with error code on test failures
- Use curl with proper headers and jq for parsing responses