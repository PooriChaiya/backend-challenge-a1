# backend-challenge-a1 — User Management API


## Run

```bash
docker compose up --build
```

Brings up MongoDB and the API on `http://localhost:8080`. Every 10s the API logs the current user count.

## Test

```bash
go test ./...
```

## Layout

```
cmd/api/                  composition root
internal/domain/          entities + domain errors (pure)
internal/port/            interfaces the service depends on
internal/service/         application service (use cases)
internal/adapter/
  mongorepo/              UserRepository → MongoDB
  bcryptpw/               PasswordHasher → bcrypt
  jwtauth/                TokenIssuer/Verifier → HS256 JWT
  httpapi/                HTTP handlers, router, middleware
```

Dependency rule: `domain` imports nothing internal; `service` imports `domain`+`port`; `adapter/*` implements `port`; `cmd/api` wires everything.

## Env vars

| Var          | Default                       |
|--------------|-------------------------------|
| `MONGO_URI`  | `mongodb://localhost:27017`   |
| `MONGO_DB`   | `usersdb`                     |
| `JWT_SECRET` | `devsecret`                   |
| `PORT`       | `8080`                        |

## Endpoints

| Method | Path            | Auth | Purpose            |
|--------|-----------------|------|--------------------|
| GET    | `/healthz`      | no   | health check       |
| POST   | `/register`     | no   | create user        |
| POST   | `/login`        | no   | returns JWT        |
| GET    | `/users`        | yes  | list users         |
| GET    | `/users/{id}`   | yes  | get one            |
| PUT    | `/users/{id}`   | yes  | update name/email  |
| DELETE | `/users/{id}`   | yes  | delete             |

## curl walkthrough

```bash
# register
curl -s -X POST localhost:8080/register \
  -H 'content-type: application/json' \
  -d '{"name":"Alice","email":"alice@example.com","password":"secret123"}'

# login → grab the token
TOKEN=$(curl -s -X POST localhost:8080/login \
  -H 'content-type: application/json' \
  -d '{"email":"alice@example.com","password":"secret123"}' | jq -r .token)

# list
curl -s -H "Authorization: Bearer $TOKEN" localhost:8080/users

# update
ID=$(curl -s -H "Authorization: Bearer $TOKEN" localhost:8080/users | jq -r '.[0].id')
curl -s -X PUT -H "Authorization: Bearer $TOKEN" -H 'content-type: application/json' \
  -d '{"name":"Alice R."}' localhost:8080/users/$ID

# delete
curl -s -X DELETE -H "Authorization: Bearer $TOKEN" -o /dev/null -w '%{http_code}\n' localhost:8080/users/$ID
```