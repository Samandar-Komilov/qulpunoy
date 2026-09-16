# Impl Step 01 — User Register (`POST /register`)

Goal: implement the first real endpoint end-to-end, establishing the layer pattern
(handler → service → repository) that every later endpoint copies.

Transaction contract (from README S3):

```
BEGIN
    INSERT INTO users VALUES (...)
    ON UNIQUE VIOLATION: ROLLBACK (409)
COMMIT
```

Time: ~30–40 min if Go + pgx are familiar. First endpoint, so half of it is scaffolding
you reuse forever.

---

## Decisions (settle before writing)

1. **Password**: store a bcrypt hash, never plaintext. Add `golang.org/x/crypto/bcrypt`.
2. **ID**: generate `UUID` in the service (app-side), not in SQL. Add `github.com/google/uuid`.
   Keeps the repository a dumb writer and makes IDs testable.
3. **`is_active`**: default `true` on register. No email-verification flow in scope.
4. **Uniqueness**: rely on the `username` UNIQUE constraint (migration `0001`), do NOT
   pre-check with a `SELECT` — that is a race. Insert, then map the unique violation to 409.
5. **Single-statement insert**: no explicit `BEGIN/COMMIT` needed — one `INSERT` is its own
   transaction. Use `pool.Exec` directly. Save explicit tx for the multi-step `POST /orders`.
6. **Layer ownership**:
   - repository → SQL only, returns typed errors, knows nothing about HTTP or bcrypt.
   - service → business rules (hash, uuid, validation), returns domain errors.
   - handler → HTTP only (decode, status codes, JSON), no SQL, no bcrypt.

---

## Steps

### 1. Add deps (~2 min)

```
go get github.com/google/uuid
go get golang.org/x/crypto/bcrypt
go mod tidy
```

### 2. Domain model — `internal/models/user.go` (new package)

Define the `User` entity matching the `users` table columns:

- `ID uuid.UUID`
- `Username string`
- `Password string`  // holds the hash, not plaintext
- `IsActive bool`
- `JoinedAt time.Time`
- `UpdatedAt time.Time`

This struct is the single shared type crossing all three layers. No JSON tags here —
keep request/response DTOs separate (step 5) so the wire format never leaks the hash.

### 3. Sentinel errors — `internal/models/errors.go`

Declare package-level errors the layers agree on. Keeps layers decoupled: repo returns
`ErrUsernameTaken`, service passes it up, handler maps it to a status.

- `ErrUsernameTaken` — repo detected a unique violation.

Idiom: `var ErrUsernameTaken = errors.New("username already taken")`.

### 4. Repository — `internal/repositories/user_repository.go`

- Struct `UserRepository` holding `pool *pgxpool.Pool`.
- Constructor `NewUserRepository(pool *pgxpool.Pool) *UserRepository`.
- Method:

  ```
  func (r *UserRepository) Create(ctx context.Context, u *models.User) error
  ```

Inside:

- `INSERT INTO users (id, username, password, is_active) VALUES ($1, $2, $3, $4)`.
  Let `joined_at` / `updated_at` take their column defaults — do not send them.
- Run with `r.pool.Exec(ctx, query, ...)`.
- On error, detect the unique violation and translate it:

  ```
  var pgErr *pgconn.PgError
  if errors.As(err, &pgErr) && pgErr.Code == "23505" {
      return models.ErrUsernameTaken
  }
  return fmt.Errorf("create user: %w", err)   // wrap everything else
  ```

  `23505` = `unique_violation`. Import `github.com/jackc/pgx/v5/pgconn`.

> Why translate here: only the repo knows it's Postgres. Upper layers must not import
> pgconn or know error codes.

### 5. Request/response DTOs — in the handler file (step 6) or `internal/routers/dto.go`

- `registerRequest{ Username string \`json:"username"\`; Password string \`json:"password"\` }`
- `registerResponse{ ID string \`json:"id"\`; Username string \`json:"username"\` }`

Never put `Password` in any response DTO.

### 6. Service — `internal/services/user_service.go`

- Struct `UserService` holding `repo *repositories.UserRepository`.
- Constructor `NewUserService(repo *repositories.UserRepository) *UserService`.
- Method:

  ```
  func (s *UserService) Register(ctx context.Context, username, password string) (*models.User, error)
  ```

Steps inside:

1. Validate input: non-empty username, password length (e.g. `>= 8`). Return a validation
   error (add `ErrInvalidInput` sentinel) — do NOT reach the DB on bad input.
2. `hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)`.
3. Build the `User`: `ID: uuid.New()`, `Password: string(hash)`, `IsActive: true`.
4. `s.repo.Create(ctx, u)` — return its error unchanged (so `ErrUsernameTaken` propagates).
5. Return the `user`.

### 7. Handler — `internal/routers/user_handler.go`

- Struct `UserHandler` holding `service *services.UserService`.
- Constructor `NewUserHandler(service *services.UserService) *UserHandler`.
- Method `func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request)`:

  1. Decode `registerRequest` from `r.Body`. On decode error → `400`.
  2. Call `h.service.Register(r.Context(), req.Username, req.Password)`.
  3. Map errors → status:
     - `ErrInvalidInput` → `400`
     - `ErrUsernameTaken` → `409`
     - default → `500` (log the real error with `slog`, return a generic message).
  4. Success → `201 Created`, JSON `registerResponse`.

Add a tiny JSON helper (e.g. `writeJSON(w, status, v)` and `writeError(w, status, msg)`)
in `internal/routers/response.go`. You will reuse it in every handler — write it once now.

### 8. Wire routes — `internal/routers/router.go`

Add a constructor that builds the chi router and takes the handlers it needs:

```
func NewRouter(userHandler *UserHandler) *chi.Mux {
    r := chi.NewRouter()
    r.Use(middleware.Logger)
    r.Post("/register", userHandler.Register)
    return r
}
```

Move the `chi.NewRouter()` setup out of `main.go` into here so `main` only wires
dependencies, not routes.

### 9. Compose in `cmd/server/main.go`

After `pool` is created, build the chain top-down and hand the router to the server:

```
userRepo := repositories.NewUserRepository(pool)
userService := services.NewUserService(userRepo)
userHandler := routers.NewUserHandler(userService)
r := routers.NewRouter(userHandler)
```

Replace the inline `r := chi.NewRouter()` block with `r := routers.NewRouter(userHandler)`.

---

## Verify

1. `make run`
2. Create a user:

   ```
   curl -i -X POST localhost:8080/register \
     -H 'Content-Type: application/json' \
     -d '{"username":"sam","password":"password123"}'
   ```

   Expect `201` + `{"id":"...","username":"sam"}`.

3. Repeat the same command → expect `409`.
4. `curl -i -X POST ... -d '{"username":"x","password":"short"}'` → expect `400`.
5. Check DB: `SELECT id, username, password, is_active FROM users;` — password must be a
   bcrypt hash (`$2a$...`), `is_active = true`.

---

## Files touched

New:
- `internal/models/user.go`
- `internal/models/errors.go`
- `internal/repositories/user_repository.go`
- `internal/services/user_service.go`
- `internal/routers/user_handler.go`
- `internal/routers/response.go`
- `internal/routers/router.go`

Edited:
- `cmd/server/main.go`
- `go.mod` / `go.sum`

Next: `POST /token` (JWT issue) + auth middleware — reuses this same layering.
