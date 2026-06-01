---
name: ssh-mux
description: >
  SSH ControlMaster lifecycle manager for FIDO2/YubiKey SSH keys. Use when
  the user mentions ssh-mux, SSH multiplexing, ControlMaster connections,
  or when troubleshooting repeated YubiKey/FIDO2 PIN prompts during SSH
  operations. Also use when encountering SSH mux-related socket errors or
  when tools (git, rsync, ansible) need shared SSH connections.
---

# ssh-mux

Manages SSH ControlMaster connections so FIDO2 keys only require one auth.

## Commands

- `ssh-mux init` — one-time setup, writes `~/.ssh/mux.conf` and socket dir
  - `--force` — overwrite existing `mux.conf`
  - `--no-edit` — skip modifying `~/.ssh/config` (for externally managed configs like home-manager)
- `ssh-mux up <user@host>` — establish a master connection (interactive, needs terminal)
  - `-p <port>` — non-standard SSH port
- `ssh-mux down [user@host]` — tear down one or all masters
- `ssh-mux status [user@host]` — check active masters

Destinations must be `user@host` format.

## Agent limitations

`ssh-mux up` requires terminal stdin for PIN/touch prompts — agents cannot run it.
Agents can run `status`, `down`, and `init --no-edit`.

If a user asks to bring up a connection, provide the command for them to run manually.

## Socket path

Sockets live at `~/.ssh/sockets/<user>@<host>-<port>` (e.g. `~/.ssh/sockets/git@github.com-22`).

## Common issues

- **"mux.conf is not included in your SSH config"** — the user ran `init --no-edit` but hasn't added `Include ~/.ssh/mux.conf` to their SSH config
- **"Session open refused by peer"** — SSH server's `MaxSessions` (default 10) exceeded; reduce parallelism
- **Stale sockets** — `ssh-mux down` cleans up; stale socket files in `~/.ssh/sockets/` can also be removed manually
