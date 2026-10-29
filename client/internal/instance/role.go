package instance

import "fmt"

// Role decides which installation wins when two want the same machine-wide
// resource: the default route, catch-all DNS, an identical route.
type Role string

const (
	// RolePrimary behaves as a lone installation always has.
	RolePrimary Role = "primary"
	// RoleSecondary never takes the default route or system DNS and loses
	// identical routes.
	RoleSecondary Role = "secondary"
)

// ParseRole parses a role given on the command line. Empty input is secondary.
// Anything unknown is secondary too, the more restrictive role, and is reported
// as an error so the caller can log it.
func ParseRole(s string) (Role, error) {
	switch Role(asciiLower(s)) {
	case RolePrimary:
		return RolePrimary, nil
	case RoleSecondary, "":
		return RoleSecondary, nil
	default:
		return RoleSecondary, fmt.Errorf("unknown installation role %q, using %s", s, RoleSecondary)
	}
}
