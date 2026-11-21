# Hexfetch {{version}}

{{summary}}

## Downloads

Static binaries without dependencies. Pick the file for your platform in the Assets section of this release.

| Target | Platform |
| --- | --- |
| `linux-amd64` | 64-bit x86 Linux |
| `linux-arm64` | 64-bit ARM Linux |
| `linux-armv7` | 32-bit ARM Linux (ARMv7) |
| `linux-386` | 32-bit x86 Linux |
| `linux-riscv64` | 64-bit RISC-V Linux |
| `android-arm64` | Android arm64 (Termux) |

## Checksums

{{assets}}

## Install

1. Download the binary for your platform and `SHA256SUMS` into one directory.
2. Verify: `sha256sum -c SHA256SUMS --ignore-missing`
3. Make it executable and put it in your `PATH`: `chmod +x hexfetch_* && mv hexfetch_* ~/.local/bin/hexfetch`

## Licenses

`LICENSE` and `THIRD_PARTY_LICENSES` are attached to this release. The same texts are embedded in the binary: run `hexfetch --licenses`.

## Quick start

    hexfetch
    hexfetch --gen-config
    hexfetch --help

`--gen-config` creates the default YAML config and prints its location. Details are in the README.

## Changes

Full comparison: {{compare}}

<details>
<summary>Commits</summary>

{{changes}}

</details>

Released {{date}} as {{tag}}.
