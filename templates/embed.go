package templates

import "embed"

// AuditTemplate contains the embedded team-audit template tree.
//
//go:embed all:audit
var AuditTemplate embed.FS

// RoleSessionTemplate contains the embedded single-role session template tree.
//
//go:embed all:role-session
var RoleSessionTemplate embed.FS
