package templates

import "embed"

// AuditTemplate contains the embedded team-audit template tree.
//
//go:embed all:audit
var AuditTemplate embed.FS
