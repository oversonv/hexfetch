# Known problems: paths, configuration standards, error handling

Static review of v0.5.10. "Fixed" items were changed in this patch set; nothing
here was verified on other distributions yet. Go tests must be run by the
maintainer (`go test ./...`).

| # | Severity | Area | Problem | Where | Status |
|---|----------|------|---------|-------|--------|
| 1 | Medium | Paths | Modules bypass the redirectable `procRoot`/`sysRoot`/`etcRoot` with literals, so fixture tests cannot cover them | `uptime`, `wm`, `kernel_linux`, `title`, `initsystem` | Fixed: now use `procPath`; guard test `TestNoHardcodedProcOrSysPaths` added |
| 2 | Low | Paths | Remaining literals outside the roots: `/etc` (`osage`, `initsystem` checks), `/var/run/utmp`, `/run/*`, `/var/lib/*`, `/dev/tty` | `osage.go`, `users.go`, `packages.go`, `initsystem.go`, `termwidth_linux.go` | Open: accepted for now, guard test covers `/proc` and `/sys` only |
| 3 | Medium | Config dirs | Three different rules: `hexfetch.yaml` in `$HOME/hex` (no XDG), `theme`/`terminalfont` via `os.UserConfigDir`, `wmtheme`/`wallpaper` hardcoded `$HOME/.config` | `config.go`, `wmtheme.go`, `wallpaper.go` | Partly fixed: `wmtheme`/`wallpaper` use `userConfigFile` (honors `XDG_CONFIG_HOME`). `DefaultPath` unchanged: compatibility decision for the maintainer |
| 4 | Low | Docs | README told users to copy `hex/hexfetch.yaml`, which no longer exists | `README.md`, `AUDIT.md` | Fixed: profile moved to `docs/termux-profile.md`, note added to `AUDIT.md` |
| 5 | Low | Errors | `command` duplicated the 3 s timeout instead of `commandTimeout` | `command.go` | Fixed |
| 6 | Medium | Packages | Gentoo (`/var/db/pkg`) and Slackware (`/var/log/packages`) not detected, output says "no package manager found" | `packages.go` | Fixed: `portage` and `pkgtool` counters + tests |
| 7 | Medium | Errors | Three vocabularies: `Unsupported` (57 uses), `wrapUnavailableError` (7), raw errors (`command` returns the bare `exec` error) | `modules/*` | Open: needs a decision on user-visible wording |
| 8 | Medium | Errors | `countRPM` returns 0 on timeout or failure (silent undercount); runs `rpm -qa` on any system that has `/var/lib/rpm` | `packages.go` | Open |
| 9 | Low | Errors | About 35 swallowed errors (`_ :=`) in modules | `modules/*` | Open: needs per-site review |
| 10 | Low | Portability | utmp record size fixed at 384 bytes; musl may use 400 (hypothesis, unverified). `$USER` fallback keeps the output non-empty | `users.go` | Open: verify on Alpine |
| 11 | Medium | Packages | NixOS: only profile manifests are counted, system packages (`/run/current-system/sw`) are not | `packages.go` | Open |
| 12 | Medium | Termux | Prefix hardcoded to `/data/data/com.termux`; forks with another app id are not found; Android 10+ hides other processes' `/proc/<pid>/comm` (`wm`, `initsystem`) | `packages.go`, `wm.go`, `initsystem.go` | Open |
| 13 | Low | Portability | In containers PID 1 `comm` (`tini`, `bash`) can be printed as the init system | `initsystem.go` | Open |
| 14 | Low | Config | Several formats coexist: YAML (current), legacy TOML/JSONC (migration only), and `hexfetch.toml` (embedded package metadata, not user config) | `config.go`, `cmd/hexfetch/meta.go` | Open: document the distinction |
| 15 | Low | Determinism | `wm` scans `/proc` in directory order (lexicographic PIDs), so with several window managers the pick is arbitrary | `wm.go` | Open |

Suggested next step: run `strace -f -e trace=openat,stat,execve ./hexfetch` in
Alpine, Debian, Fedora, Arch, Void, Gentoo, NixOS images and on Termux, then
compare the `ENOENT`/`EACCES` paths with this table.
