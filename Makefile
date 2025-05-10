# Makefile at root of ottoapp

.PHONY: build backend frontend dev

# Build React frontend and Go binary
build: frontend backend

# Build frontend (React with Tailwind)
frontend:
	cd ottofe && npm install && npm run build

# Cross-compile Go backend for Linux (x86_64)
backend:
	cd ottobe && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/boxdb

# Dev mode: runs both frontend and backend locally
dev:
	@echo "Starting Go backend and React frontend..."
	@echo make -j2 run-backend run-frontend

run-backend:
	cd ottobe && go build && ./ottobe --host localhost --port 29631 --data ../userdata

run-frontend:
	cd ottofe && npm run dev

