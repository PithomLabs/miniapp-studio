## prompt


Determination complete. I verified everything observable from here rather than asking.

  Finding: no local PostgreSQL provision currently exists

  Evidence (all read-only, sandbox-observed): no server package installed (only PG16 client tools), no postgres/initdb/pg_ctl
  binaries anywhere on PATH, no /var/lib/postgresql, /etc/postgresql, or socket files, no PGDG apt source (Ubuntu 24.04 repos
  top out at PG16, so PG19 isn't installable without it), and Docker state is unobservable from my side. The one thing that
  persists is the /tmp/boundary-b/owner.dsn convention file (present, 0600, values untouched). Net effect: Phase 1 is blocked
  with cause identified — there is nothing to verify yet, so per the fail-closed rule I am not claiming or skipping anything.

  Two things I deliberately ruled out rather than attempting: user-space source builds and static-binary distributions would
  violate the verified-build parity doctrine (wrong provenance, behavior risk) — worse than blocked.

  Resolution: native PG19 provisioning (owner-terminal, one time)

  This needs root, so it's yours to run; everything after it is mine. Exact procedure:

    sudo apt update && sudo apt install -y postgresql-common
    sudo /usr/share/postgresql-common/pgdg/apt.postgresql.org.sh -y
    sudo apt update && sudo apt install -y postgresql-19
    pg_createcluster 19 miniapp --port 5434 --start
    pg_lsclusters                              # expect: 19/miniapp online on 5434
    sudo -u postgres psql -c "SHOW server_version;"   # expect 19.x

  Then maintainer role + credential (generate fresh, 0600 file, never paste the value anywhere):

    DBPASS=$(openssl rand -hex 24)
    sudo -u postgres psql -c "CREATE ROLE studio_maint WITH LOGIN PASSWORD '$DBPASS' CREATEDB CREATEROLE;"
    printf '%s' "postgres://studio_maint:${DBPASS}@127.0.0.1:5434/postgres" > /tmp/boundary-b/owner.dsn
    chmod 600 /tmp/boundary-b/owner.dsn
    unset DBPASS

  Loopback binding is Debian's default (localhost); if your postgresql.conf differs, keep it loopback-only. miniapp-dev-db-1
  is untouched by all of this (and nothing here needs Docker back).

  What happens after

  Paste back only: pg_lsclusters output, SHOW server_version; output, and confirmation the DSN file was written (not its
  contents). I will then run the full Phase 1 batch myself where possible (studio dev doctor/fingerprint/init/suites via
  owner-terminal commands I supply), update prompt28_plan.md Phase 1/2 and the stale pg_muse.md §2.3/§3.1 lines against
  verified reality, and stop before Phase 2 as authorized.

  Related notes: CI's postgres service is unaffected (still the digest-pinned PG19 image). Toolchain-image and Testcontainers
  tests cannot run in this sandbox with no local runtime — they stay CI-gated, and I'll record that in the parity report
  rather than claim them locally.

Two notes: 1) I do not need to run CLI commands just to install Postgres locally, it must be specified as a command in Taskfile as pre-requisite to setup and 2) I am questioning the value of CI here, I believe CI can be done via Sprites without resorting to GitHub Actions, is that right?


## answer

