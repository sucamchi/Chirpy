# Chirpy

Chirpy is a small Twitter-like REST API written in Go. Users can sign up, log in, post short messages ("chirps") of up to 140 characters, and browse them. It uses PostgreSQL for storage, JWT access tokens with revocable refresh tokens for authentication, and a webhook that upgrades users to **Chirpy Red** after payment.

## Features

- User registration, login and profile updates (passwords hashed with Argon2id)
- JWT access tokens (1 hour) and revocable refresh tokens (60 days)
- Create, read, list and delete chirps, with ownership checks on deletion
- Profanity filter (`kerfuffle`, `sharbert`, `fornax` become `****`)
- Filtering by author and sorting by creation time when listing chirps
- API-key protected Polka webhook for Chirpy Red upgrades
- Type-safe SQL via [sqlc](https://sqlc.dev) and migrations via [goose](https://github.com/pressly/goose)

## Requirements

- Go 1.27 or newer (see `go.mod`)
- PostgreSQL
- [goose](https://github.com/pressly/goose) (migrations), and optionally [sqlc](https://sqlc.dev) (only to regenerate query code)

## Getting started

```bash
git clone https://github.com/sucamchi/Chirpy.git
cd Chirpy
```

Create a database, then add a `.env` file in the project root:

```env
DB_URL="postgres://user:password@localhost:5432/chirpy?sslmode=disable"
PLATFORM="dev"
JWT_SECRET="a-long-random-string"
POLKA_KEY="the-api-key-polka-gave-you"
```

| Variable     | Description                                                          |
| ------------ | -------------------------------------------------------------------- |
| `DB_URL`     | PostgreSQL connection string                                         |
| `PLATFORM`   | Set to `dev` to allow `POST /admin/reset`                            |
| `JWT_SECRET` | Secret used to sign access tokens (generate one with `openssl rand -base64 64`) |
| `POLKA_KEY`  | API key Polka must send to the webhook endpoint                      |

Run the migrations and start the server:

```bash
goose -dir sql/schema postgres "$DB_URL" up
go run .
```

The server listens on `http://localhost:8080`.

## API

Authenticated endpoints expect `Authorization: Bearer <token>`. Errors are returned as `{"error": "message"}`.

### Users and authentication

| Method | Endpoint       | Auth                | Description                                      |
| ------ | -------------- | ------------------- | ------------------------------------------------ |
| POST   | `/api/users`   | -                   | Create a user (`email`, `password`)              |
| PUT    | `/api/users`   | Access token        | Update the current user's email and password     |
| POST   | `/api/login`   | -                   | Returns the user, an access token and a refresh token |
| POST   | `/api/refresh` | Refresh token       | Exchange a refresh token for a new access token  |
| POST   | `/api/revoke`  | Refresh token       | Revoke a refresh token                           |

### Chirps

| Method | Endpoint                | Auth         | Description                              |
| ------ | ----------------------- | ------------ | ---------------------------------------- |
| POST   | `/api/chirps`           | Access token | Create a chirp (`body`, max 140 chars)   |
| GET    | `/api/chirps`           | -            | List chirps                              |
| GET    | `/api/chirps/{chirpID}` | -            | Get a single chirp                       |
| DELETE | `/api/chirps/{chirpID}` | Access token | Delete your own chirp (`403` otherwise)  |

`GET /api/chirps` supports optional query parameters:

- `author_id` - only return chirps by this user (filtered in the database)
- `sort` - `asc` (default) or `desc`, by `created_at`

```bash
curl "http://localhost:8080/api/chirps?author_id=<user-uuid>&sort=desc"
```

Invalid `author_id` or `sort` values return `400`.

### Webhooks

| Method | Endpoint              | Auth                      | Description                                  |
| ------ | --------------------- | ------------------------- | -------------------------------------------- |
| POST   | `/api/polka/webhooks` | `Authorization: ApiKey <POLKA_KEY>` | Handles `user.upgraded` events to grant Chirpy Red |

Requests without the correct API key receive `401`.

```json
{
  "event": "user.upgraded",
  "data": { "user_id": "3311741c-680c-4546-99f3-fc9efac2036c" }
}
```

### Admin and utility

| Method | Endpoint         | Description                                          |
| ------ | ---------------- | ---------------------------------------------------- |
| GET    | `/api/healthz`   | Readiness check                                      |
| GET    | `/admin/metrics` | Number of visits to `/app/`                          |
| POST   | `/admin/reset`   | Reset metrics and delete all users (`PLATFORM=dev` only) |

## Project structure

```
.
├── main.go              # configuration, routing, admin endpoints
├── handlers_users.go    # users, login, tokens, Polka webhook
├── handlers_chirps.go   # chirp endpoints
├── json.go              # response helpers and auth helper
├── internal/
│   ├── auth/            # password hashing, JWTs, refresh tokens, header parsing
│   └── database/        # sqlc-generated code
├── sql/
│   ├── schema/          # goose migrations
│   └── queries/         # sqlc queries
└── sqlc.yaml
```

## Development

```bash
go test ./...                 # run the tests
sqlc generate                 # regenerate internal/database after editing sql/queries
goose -dir sql/schema postgres "$DB_URL" down   # roll back the latest migration
```
