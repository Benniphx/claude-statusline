package cost

import (
	"github.com/Benniphx/claude-statusline/core/ports"
)

// ResolveSession returns a session ID and whether it's stable.
func ResolveSession(plat ports.PlatformInfo) (string, bool) {
	return plat.GetStableSessionID()
}
