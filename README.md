# BugYou!! Backend

Go Gin backend for the BugYou!! MVP: users can report bugs or requirements, track tickets, and developers can manage issue status with mandatory comments and old pending reminders.

## Stack

- Go 1.24
- Gin
- MongoDB
- JWT authentication
- Local file uploads

## Setup

```sh
cp .env.example .env
GOWORK=off go mod tidy
GOWORK=off go run ./cmd/api
```

The API runs on `http://localhost:8080` by default.

## API

Auth:

- `POST /api/auth/register`
- `POST /api/auth/login`
- `GET /api/auth/me`

User:

- `POST /api/issues`
- `GET /api/issues/my`
- `GET /api/issues/track/:ticketId`

Developer:

- `GET /api/developer/dashboard?filter=all&product=ZenClass`
- `GET /api/developer/issues?status=Open&priority=High&product=ZenClass&type=Bug`
- `PATCH /api/developer/issues/:id/status`
- `GET /api/developer/reminders`

## Product Options

The MVP product list is:

- ZenClass
- Classify
- Hyernet
- PlacementInfo
- GuviPortal
- Other

`Other` is temporary and can be replaced later in `internal/constants/options.go`.
