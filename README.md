# ssh-mux

SSH ControlMaster lifecycle manager. Authenticate once with your
FIDO2/YubiKey, reuse the connection everywhere.

## Why

FIDO2 SSH keys (ED25519-SK) require PIN + physical touch for every new
connection. Tools that spawn multiple SSH sessions — git, ansible, rsync
batches — either fail or demand repeated interaction.

SSH's ControlMaster multiplexing solves this, but requires manual config and
obscure flags. `ssh-mux` wraps the lifecycle into three commands.

## Install

```bash
go install github.com/Otard95/ssh-mux@latest
```

<details>
<summary>Nix</summary>

```bash
# run directly
nix run github:Otard95/ssh-mux -- up git@github.com

# install to profile
nix profile add github:Otard95/ssh-mux

# build locally
nix build
./result/bin/ssh-mux
```

</details>

## Usage

### Initialize

One-time setup — creates `~/.ssh/mux.conf` and wires it into your SSH config:

```bash
ssh-mux init
```

If your SSH config is managed externally (e.g. NixOS home-manager), use
`--no-edit` to skip modifying `~/.ssh/config`:

```bash
ssh-mux init --no-edit
```

Then add the following to your SSH config yourself:

```
Include ~/.ssh/mux.conf
```

### Open a master connection

```bash
ssh-mux up git@github.com
# Prompts for YubiKey PIN + touch once
# Master connection backgrounds after auth
```

Non-standard port:

```bash
ssh-mux up git@my-server.com -p 2222
```

### Check status

```bash
ssh-mux status                    # list all active masters
ssh-mux status git@github.com    # check a specific destination
```

### Tear down

```bash
ssh-mux down git@github.com   # close specific master
ssh-mux down                   # close all masters
```

## How it works

`ssh-mux init` writes an SSH config snippet:

```
Host *
    ControlMaster auto
    ControlPath ~/.ssh/sockets/%r@%h-%p
```

`ControlMaster auto` means: be a master if no socket exists, otherwise reuse
the existing one. This is inert when no master is running — SSH falls back to
normal connections.

`ssh-mux up` establishes a master connection that backgrounds after
authentication. All subsequent SSH connections to that destination
automatically multiplex through it — no per-tool integration needed.

## Concurrency limits

SSH servers limit multiplexed sessions via `MaxSessions` (default: 10).
Callers doing parallel work should stay below this threshold. `ssh-mux` does
not enforce concurrency limits.
