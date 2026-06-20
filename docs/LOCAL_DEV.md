# Local Development Setup

How to spin up Lunartica (Go API + PostgreSQL + Next.js web) on a fresh machine
for development and for testing the LunarWing `multica-bridge` interop.

> ⚠️ **Local dev only.** These instructions enable a passwordless login bypass
> (`MULTICA_DEV_VERIFICATION_CODE`) and use throwaway credentials. Never use this
> setup, the dev code, or `APP_ENV=development` on a public/production instance.

## Prerequisites

- **git**
- **Go ≥ 1.26** — or any Go; set `GOTOOLCHAIN=auto` and it auto-fetches the
  version pinned in `server/go.mod`.
- **Node ≥ 20** (22 recommended). pnpm comes via corepack: `corepack enable pnpm`.
- **Docker** *or* **Podman** (for the PostgreSQL container).

## 1. Clone + configure

```bash
git clone https://github.com/LunarWingOrg/Lunartica.git
cd Lunartica
git checkout lunarpunk-reskin          # the lunarpunk reskin branch

cp .env.example .env
sed -i "s/^JWT_SECRET=.*/JWT_SECRET=$(openssl rand -hex 32)/" .env
echo "MULTICA_DEV_VERIFICATION_CODE=424242" >> .env   # passwordless dev login
corepack enable pnpm && pnpm install
```

Defaults in `.env` already point at Postgres `localhost:5432` (db/user/pass all
`multica`), the API on `:8080`, and the web app on `:3000`.

## 2a. Run it — Docker (simplest)

```bash
export GOTOOLCHAIN=auto      # only needed if local Go < 1.26
make start                   # Postgres (compose) + migrate + API + web together
```

`make start` brings up the Postgres container via `docker compose`, applies
migrations, then runs the Go API (`:8080`) and the Next.js frontend (`:3000`).
Open <http://localhost:3000>.

(`make dev` does the same plus first-run env bootstrap; `make stop` stops the app
processes; `make db-down` stops the database container.)

## 2b. Run it — Podman / no Docker daemon

```bash
# Postgres (pgvector)
podman run -d --name lunartica-pg -p 127.0.0.1:5432:5432 \
  -e POSTGRES_DB=multica -e POSTGRES_USER=multica -e POSTGRES_PASSWORD=multica \
  docker.io/pgvector/pgvector:pg17

# migrate + run the API (loads .env; fetches Go 1.26 toolchain if needed)
set -a; . ./.env; set +a
( cd server && GOTOOLCHAIN=auto go run ./cmd/migrate up )
( cd server && GOTOOLCHAIN=auto APP_ENV=development go run ./cmd/server ) &

# run the web frontend
NEXT_PUBLIC_API_URL=http://localhost:8080 NEXT_PUBLIC_WS_URL=ws://localhost:8080/ws pnpm dev:web
```

Open <http://localhost:3000>.

## 3. Log in

Enter **any email** → on the code screen enter **`424242`**. The dev bypass is
active because `APP_ENV` ≠ `production` **and** `MULTICA_DEV_VERIFICATION_CODE` is
set. Without the dev code, read the real 6-digit code from the DB (Resend is unset
so no email is sent):

```bash
podman exec lunartica-pg psql -U multica -d multica -t \
  -c "select code from verification_codes order by created_at desc limit 1;"
# (docker: docker exec <postgres-container> psql ... )
```

## 4. Accessing from another machine (headless/remote box)

The dev server is plain HTTP; browsing the box's LAN IP can trip HTTPS-Only mode
and the client JS may not hydrate. Tunnel instead, from your laptop:

```bash
ssh -L 3000:localhost:3000 -L 8080:localhost:8080 user@that-machine
```

Then browse <http://localhost:3000> — it's localhost on your side, so the JS
hydrates and the API (`:8080`) is reachable through the tunnel.

## 5. Optional — test the LunarWing bridge against this instance

The interop scripts live in the LunarWing repo:

```bash
git clone https://github.com/LunarWingOrg/lunarwing.git
cd lunarwing && git checkout 1.1.5-lunartica-1

# With Lunartica running on :8080, replay the bridge's daemon-protocol calls:
MULTICA_DEV_CODE=424242 ic/scripts/multica-bridge-smoke-test.sh
```

Or stand up Lunartica from scratch in one command:

```bash
LUNARTICA_DIR=/path/to/Lunartica ic/scripts/multica-local-test-server.sh up
# ... and tear it down:
ic/scripts/multica-local-test-server.sh down
```

## Notes

- **Ports:** API `8080`, web `3000`, Postgres `5432` (override via `.env`).
- **Go toolchain:** the repo pins Go 1.26.1 in `server/go.mod`; `GOTOOLCHAIN=auto`
  lets an older local Go download it automatically.
- **Teardown (Podman path):** `podman rm -f lunartica-pg` and stop the `go run` /
  `pnpm dev:web` processes (e.g. `kill` them or close the terminal).
