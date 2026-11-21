// Package legal embeds the license texts that ship inside every binary.
package legal

import _ "embed"

// License is the hexfetch license (MIT).
//
//go:embed LICENSE
var License string

// ThirdParty holds the notices of bundled third-party code and data.
//
//go:embed THIRD_PARTY_LICENSES
var ThirdParty string
