# Legal and License Audit

Audit date: 2026-11-22  
Project version reviewed: v0.6.3  
Scope: current repository contents, bundled assets, Go module dependencies, license notices, and the removal of previously uncertain ARM lookup data and NixOS logo adaptation.

This is a technical provenance and compliance review, not a legal opinion. The applicable rules depend on the jurisdictions in which the project and its binaries are distributed. This review does not certify every logo trademark or provide a jurisdiction-by-jurisdiction legal opinion.

## Executive assessment

The repository's own Go application code is declared under MIT. The bundled fastfetch/neofetch-derived logo assets and Go dependencies retain their separate upstream notices in `THIRD_PARTY_LICENSES`. Those notices are also embedded in the executable and checked against the root copies by `src/legal/licenses_test.go`.

The two previously identified provenance concerns have been removed from the current source tree:

- `src/modules/cpuparts.go` and its static ARM implementer/part-name lookup tables have been deleted. CPU identification now uses descriptive fields supplied by the running Linux kernel in `/proc/cpuinfo`; the application no longer ships a copied ARM identifier-to-name database.
- `src/logo/ascii/n/nixos_average.txt`, its CC BY 4.0 attribution block, its alias entries, and its dedicated gradient renderer/configuration have been removed. The default logo is now selected automatically from `/etc/os-release`, with the generic `linux` logo as fallback.

This avoids distributing those specific materials. It does not establish that every other third-party asset is free of all copyright or trademark restrictions; the remaining fastfetch/neofetch notices and identification-only use should be preserved.

## License inventory

| Component | Declared or identified license | Review result |
| --- | --- | --- |
| Hexfetch application code | MIT, root `LICENSE` | Notice present |
| Fastfetch-derived logo collection and metadata | MIT | Copyright and license text included in `THIRD_PARTY_LICENSES` |
| Neofetch-origin logos incorporated through fastfetch | MIT | Copyright and license text included in `THIRD_PARTY_LICENSES` |
| `github.com/clipperhouse/uax29/v2 v2.2.0` | MIT | Notice included |
| `github.com/mattn/go-runewidth v0.0.31` | MIT | Notice included |
| `github.com/pelletier/go-toml/v2 v2.4.3` | MIT | Notice included |
| `github.com/spf13/pflag v1.1.0` | BSD 3-Clause | Notice included |
| `gopkg.in/yaml.v3 v3.0.1` | MIT for the identified libyaml-derived files; Apache-2.0 for the remaining project files | Both license notices included; full Apache-2.0 text included |
| Linux CPU identification | Runtime fields read from `/proc/cpuinfo` | No static third-party ARM name table is shipped |

The dependency inventory is based on the versions in `go.mod` and the dependency license texts included in `THIRD_PARTY_LICENSES`. `go.sum` may contain checksums for transitive or test dependencies that are not compiled into the application; those are not automatically runtime components.

## ARM CPU identification

The previous static mappings from ARM implementer and part identifiers to vendor/core names have been removed rather than copied, paraphrased, or rebuilt from another database. The CPU module now prioritizes the kernel-provided `model name`, `Hardware`, and `Processor` fields. If none is present, it does not fabricate a model name from the numeric implementer/part IDs.

This is a deliberate source-provenance decision. The upstream util-linux `sys-utils/lscpu-arm.c` file is marked GPL-2.0-or-later and documents that its data has multiple sources. The former project table was not necessary to display the kernel's own CPU description, so it has been removed entirely. No util-linux C implementation or ARM mapping table is included in the current source tree.

Relevant upstream reference for the removed source relationship: [util-linux ARM lscpu source](https://github.com/util-linux/util-linux/blob/master/sys-utils/lscpu-arm.c).

## Logo artwork and trademarks

The custom NixOS-derived `nixos_average` adaptation is no longer included. Its image file, CC BY 4.0 attribution block, custom gradient-band parser, and dedicated `logo_colors` configuration/CLI interface have been removed. The remaining NixOS distribution logos, like the rest of the distribution logo collection, are the fastfetch-derived assets covered by the upstream notices in `THIRD_PARTY_LICENSES`.

Copyright licenses for logo artwork do not automatically grant trademark rights. Distribution names and logos are included for operating-system identification only and do not imply affiliation or endorsement. A particular use in product branding, paid promotion, or merchandise may require additional review or permission from the relevant rights holder.

## Distribution checklist

- Keep the root `LICENSE` and `THIRD_PARTY_LICENSES` in source archives and release assets.
- Keep the embedded copies under `src/legal/` synchronized with the root files. A test verifies byte-for-byte equality.
- Preserve the MIT and BSD notices and the MIT/Apache-2.0 notices for the Go dependencies when redistributing binaries or source archives.
- Do not claim that a copyright license grants rights to third-party trademarks.
- Do not reintroduce external lookup tables or artwork without recording their source and license terms.

## Scope limitations

This review does not provide a complete country-by-country trademark analysis or an individual provenance record for every imported logo file. It also does not determine whether a particular distribution's trademark policy permits every possible use. Those questions require jurisdiction-specific legal review and, where relevant, verification of the individual asset's upstream history.
