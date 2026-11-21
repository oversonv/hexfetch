package logodata

import "embed"

//go:embed ascii colors.txt aliases.txt
var FS embed.FS
