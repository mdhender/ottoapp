# AGENT.md

## Commands
- Build: `go build`
- Run server: `./ottoapp serve --database /path/to/db --host localhost --port 29631`
- Build for Linux: `GOOS=linux GOARCH=amd64 go build -o ottoapp.exe`
- Tests: `go test ./...`
- Run single test: `go test -v ./path/to/package -run TestName`
- Format code: `go fmt ./...`
- Update CSS: `npx tailwindcss -i assets/css/tailwind-input.css -o assets/css/tailwind.css --watch`

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

## Architecture Changes
- Transitioning from Go HTML templates to React + Vite + Tailwind frontend
- Frontend served by `npm run dev` in development, Nginx in production
- Backend updated to serve only API endpoints
- Frontend connects to https://localhost:29631/api/... in development
- In production, frontend connects to /api/... with Nginx proxy to backend