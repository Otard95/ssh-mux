# ssh-mux — SSH ControlMaster lifecycle manager

A CLI tool that manages SSH ControlMaster connections, making FIDO2/YubiKey
SSH keys practical for tools that run many SSH operations (parallel or
sequential). Authenticate once, reuse everywhere.

## Problem

FIDO2 resident SSH keys (ED25519-SK) require PIN entry and physical touch for
each new SSH connection. Tools that spawn multiple SSH sessions (git
operations, ansible, rsync batches, etc.) either fail or demand repeated
interaction with the hardware token.

SSH's ControlMaster multiplexing solves this — one authenticated connection,
many sessions — but requires manual SSH config and obscure flags to manage.

## Solution

`ssh-mux` wraps the ControlMaster lifecycle into three commands: `init`, `up`,
and `down`. After a one-time `init`, any tool using SSH transparently benefits
from multiplexed connections without per-tool integration.

## Commands

### `ssh-mux init`

One-time setup. Configures SSH to support multiplexing.

1. Create `~/.ssh/sockets/` directory (mode 0700)
2. Write `~/.ssh/mux.conf`:
   ```
   Host *
       ControlMaster auto
       ControlPath ~/.ssh/sockets/%r@%h-%p
   ```
3. Prepend `Include ~/.ssh/mux.conf` to `~/.ssh/config` (if not already present)

**Safety:**
- Check for existing `Include` of `mux.conf` before modifying
- Back up `~/.ssh/config` before editing (e.g. `~/.ssh/config.bak.<timestamp>`)
- Validate the resulting config with `ssh -G github.com` after modification
- Never overwrite an existing `mux.conf` without `--force`

### `ssh-mux up <destination>`

Establish a ControlMaster connection. Destination is the SSH target
(e.g. `git@github.com`).

```bash
ssh-mux up git@github.com
# Prompts for YubiKey PIN + touch
# Master connection backgrounds after auth
```

Under the hood:
```
ssh -M -S ~/.ssh/sockets/<destination>-22 -N -f <destination>
```

**Behavior:**
- If a master for this destination already exists, report it and do nothing
- Stdin/stdout/stderr connected to the terminal for PIN/touch prompts
- Fail with clear error if `init` hasn't been run
- Support `-p <port>` for non-standard ports

### `ssh-mux down [destination]`

Tear down ControlMaster connections.

```bash
ssh-mux down git@github.com   # tear down specific master
ssh-mux down                   # tear down all active masters
```

Under the hood:
```
ssh -S ~/.ssh/sockets/<destination>-22 -O exit <destination>
```

**Behavior:**
- Graceful shutdown — existing sessions finish, no new ones accepted
- When no destination given, iterate all sockets in `~/.ssh/sockets/`
- No error if the master is already gone

### `ssh-mux status [destination]`

Show active ControlMaster connections.

```bash
ssh-mux status
# git@github.com  active  (socket: ~/.ssh/sockets/git@github.com-22)

ssh-mux status git@github.com
# active
```

Under the hood:
```
ssh -S ~/.ssh/sockets/<destination>-22 -O check <destination>
```

## How it works

### SSH ControlMaster multiplexing

SSH multiplexing reuses a single authenticated TCP connection for multiple
sessions. The first connection (the "master") does full authentication — key
exchange, FIDO2 PIN, touch. It then creates a Unix domain socket on disk.
Subsequent SSH connections to the same host discover the socket via
`ControlPath` and tunnel through the existing connection. No new
authentication required.

The `ControlMaster auto` setting means: "be a master if no socket exists,
otherwise be a client of the existing master." This is inert when no master
is running — SSH falls back to normal standalone connections.

### Concurrency limits

SSH servers limit multiplexed sessions via `MaxSessions` (default: 10).
When exceeded, new sessions are refused with:
```
mux_client_request_session: session request failed: Session open refused by peer
```
Callers doing parallel work should limit concurrency to stay below this
threshold. This tool does not enforce concurrency limits — that's the
caller's responsibility.

## Socket path convention

Sockets live at `~/.ssh/sockets/%r@%h-%p`:
- `%r` — remote username
- `%h` — remote hostname
- `%p` — remote port

Example: `~/.ssh/sockets/git@github.com-22`

This is deterministic, so any SSH client can independently compute the path
and discover an active master.

## Reference implementation

[ngm](https://github.com/Otard95/ngm) (Nested Git Manager) implements
per-command SSH multiplexing for parallel git operations. Relevant source at
`/home/otard/dev/smb/hackday/nested-git-manager/`:

- `git/mux.go` — ControlMaster lifecycle: establish, apply to commands, teardown
- `git/remotes.go` — Extract SSH destinations from git repos
- `git/pull.go` / `git/push.go` — Usage: `InitMux()`, `ApplyToCmd()`, semaphore-based concurrency limiting

ngm takes a different approach: it manages ControlMaster connections
internally per-command via `--mux`, using `GIT_SSH_COMMAND` to point git at
specific sockets and a semaphore to limit concurrency. This works without any
SSH config but requires per-tool integration.

`ssh-mux` is the config-based alternative — transparent to all tools via
`ControlMaster auto` and a well-known `ControlPath`.

## Implementation notes

- Language: Go (cobra for CLI, consistent with ngm)
- No runtime dependencies beyond `ssh`
- Socket directory permissions must be 0700 (SSH enforces this)
- `init` modifying `~/.ssh/config` is the only "scary" operation — be
  defensive, back up, validate
- `ControlPersist` intentionally omitted from `mux.conf` — masters should
  be explicitly managed via `up`/`down`, not left lingering
