package teams

// RoleDefinition describes one role assignment for a team member.
type RoleDefinition struct {
	CodeName string
	Title    string
	Guidance string
}

// AgentConfigRoles defines role assignments for a specific agent count.
type AgentConfigRoles struct {
	AgentCount int
	Roles      []RoleDefinition
}

// AuditType defines an audit mode selectable by the user.
type AuditType struct {
	ID          string
	Name        string
	BeadPrefix  string
	Description string
	FocusAreas  []string
	RoleConfigs []AgentConfigRoles
}

// AuditTypes is the registry of supported audit modes.
var AuditTypes = []AuditType{
	{
		ID:          "perf",
		Name:        "Performance Audit",
		BeadPrefix:  "perf",
		Description: "Find bottlenecks that slow interaction, rendering, and page load.",
		FocusAreas: []string{
			"render throughput and frame stability",
			"bundle size and code splitting",
			"network waterfalls and request priority",
			"critical rendering path and hydration",
			"cache strategy for static and API assets",
			"hot path profiling for user journeys",
		},
		RoleConfigs: []AgentConfigRoles{
			{
				AgentCount: 1,
				Roles: []RoleDefinition{
					{CodeName: "alpha", Title: "Senior performance specialist", Guidance: "Own full audit scope, prioritize highest-impact bottlenecks, and deliver a sequenced remediation plan."},
				},
			},
			{
				AgentCount: 2,
				Roles: []RoleDefinition{
					{CodeName: "alpha", Title: "Senior performance specialist", Guidance: "Lead architecture analysis, triage bottlenecks by business impact, and define acceptance criteria for fixes."},
					{CodeName: "bravo", Title: "Staff performance engineer", Guidance: "Run traces and benchmarks, validate hypotheses, and provide measurement-backed implementation recommendations."},
				},
			},
			{
				AgentCount: 3,
				Roles: []RoleDefinition{
					{CodeName: "alpha", Title: "Senior performance specialist", Guidance: "Coordinate scope, synthesize findings, and sequence improvements into delivery-ready work items."},
					{CodeName: "bravo", Title: "Staff performance engineer", Guidance: "Drive instrumentation and profiling, then document regression guards for each hotspot."},
					{CodeName: "charlie", Title: "Runtime optimization specialist", Guidance: "Deep dive into rendering and runtime internals such as scheduling, hydration, and memory churn."},
				},
			},
		},
	},
	{
		ID:          "memleak",
		Name:        "Memory Leak Audit",
		BeadPrefix:  "mem",
		Description: "Detect memory retention patterns that degrade long-running sessions.",
		FocusAreas: []string{
			"event listener lifecycle cleanup",
			"detached DOM and closure retention",
			"timer and interval disposal",
			"cache growth bounds and eviction",
			"stream and subscription teardown",
			"heap snapshot diff investigation",
		},
		RoleConfigs: []AgentConfigRoles{
			{
				AgentCount: 1,
				Roles: []RoleDefinition{
					{CodeName: "alpha", Title: "Senior memory specialist", Guidance: "Lead retention-path analysis, isolate leak vectors, and propose pragmatic cleanup patterns."},
				},
			},
			{
				AgentCount: 2,
				Roles: []RoleDefinition{
					{CodeName: "alpha", Title: "Senior memory specialist", Guidance: "Prioritize user-visible leak risks and define reproducible investigation scenarios."},
					{CodeName: "bravo", Title: "Staff diagnostics engineer", Guidance: "Capture heap snapshots and allocation timelines, then map retained objects to code ownership."},
				},
			},
			{
				AgentCount: 3,
				Roles: []RoleDefinition{
					{CodeName: "alpha", Title: "Senior memory specialist", Guidance: "Coordinate findings across features and enforce cleanup standards for long-lived views."},
					{CodeName: "bravo", Title: "Staff diagnostics engineer", Guidance: "Produce high-signal reproduction traces and validate memory improvements after fixes."},
					{CodeName: "charlie", Title: "Garbage collection domain specialist", Guidance: "Analyze allocator behavior, object tenure, and runtime-specific memory semantics."},
				},
			},
		},
	},
	{
		ID:          "lighthouse",
		Name:        "Lighthouse Score",
		BeadPrefix:  "lh",
		Description: "Raise Lighthouse scores with targeted improvements across key categories.",
		FocusAreas: []string{
			"core web vitals diagnostics",
			"accessibility rule failures",
			"SEO metadata and crawlability",
			"best practices and security headers",
			"third-party script impact",
			"repeatable lab test baselines",
		},
		RoleConfigs: []AgentConfigRoles{
			{
				AgentCount: 1,
				Roles: []RoleDefinition{
					{CodeName: "alpha", Title: "Senior web quality specialist", Guidance: "Own Lighthouse strategy end to end and prioritize fixes that improve both scores and real UX."},
				},
			},
			{
				AgentCount: 2,
				Roles: []RoleDefinition{
					{CodeName: "alpha", Title: "Senior web quality specialist", Guidance: "Set category targets, identify blocking regressions, and define implementation order."},
					{CodeName: "bravo", Title: "Staff frontend engineer", Guidance: "Execute metric-specific optimizations and produce before-and-after score evidence."},
				},
			},
			{
				AgentCount: 3,
				Roles: []RoleDefinition{
					{CodeName: "alpha", Title: "Senior web quality specialist", Guidance: "Drive overall scoring plan and align quality goals with release milestones."},
					{CodeName: "bravo", Title: "Staff frontend engineer", Guidance: "Implement rendering and loading improvements with measurable metric impact."},
					{CodeName: "charlie", Title: "Search and accessibility specialist", Guidance: "Focus on SEO and accessibility category wins while preserving product semantics."},
				},
			},
		},
	},
	{
		ID:          "security",
		Name:        "Security Audit",
		BeadPrefix:  "sec",
		Description: "Identify exploitable risks and harden the application against common attacks.",
		FocusAreas: []string{
			"xss and output encoding boundaries",
			"authentication and session controls",
			"authorization checks and privilege boundaries",
			"injection vectors across data layers",
			"secrets handling and transport security",
			"owasp aligned risk prioritization",
		},
		RoleConfigs: []AgentConfigRoles{
			{
				AgentCount: 1,
				Roles: []RoleDefinition{
					{CodeName: "alpha", Title: "Senior security specialist", Guidance: "Lead threat-focused review, rank vulnerabilities by exploitability, and define mitigation plan."},
				},
			},
			{
				AgentCount: 2,
				Roles: []RoleDefinition{
					{CodeName: "alpha", Title: "Senior security specialist", Guidance: "Own threat model, coordinate remediation priorities, and set validation criteria."},
					{CodeName: "bravo", Title: "Staff application security engineer", Guidance: "Perform code-level verification, build proof-of-concept checks, and document secure alternatives."},
				},
			},
			{
				AgentCount: 3,
				Roles: []RoleDefinition{
					{CodeName: "alpha", Title: "Senior security specialist", Guidance: "Coordinate multi-surface risk analysis and package findings for rapid triage."},
					{CodeName: "bravo", Title: "Staff application security engineer", Guidance: "Validate code paths and exploitability, then propose safe refactors with tests."},
					{CodeName: "charlie", Title: "Identity and cryptography specialist", Guidance: "Deep dive on auth flows, token lifecycles, and cryptographic control correctness."},
				},
			},
		},
	},
	{
		ID:          "maint",
		Name:        "Code Maintainability",
		BeadPrefix:  "maint",
		Description: "Improve code health, readability, and long-term development velocity.",
		FocusAreas: []string{
			"cyclomatic complexity hotspots",
			"naming and abstraction clarity",
			"error handling consistency",
			"test coverage and reliability gaps",
			"module boundaries and coupling",
			"technical debt triage and payoff",
		},
		RoleConfigs: []AgentConfigRoles{
			{
				AgentCount: 1,
				Roles: []RoleDefinition{
					{CodeName: "alpha", Title: "Senior maintainability specialist", Guidance: "Own health assessment, identify high-friction patterns, and sequence refactoring recommendations."},
				},
			},
			{
				AgentCount: 2,
				Roles: []RoleDefinition{
					{CodeName: "alpha", Title: "Senior maintainability specialist", Guidance: "Define modernization priorities and align improvements with team delivery cadence."},
					{CodeName: "bravo", Title: "Staff software architect", Guidance: "Map dependency and module structure, then propose changes that reduce coupling and ambiguity."},
				},
			},
			{
				AgentCount: 3,
				Roles: []RoleDefinition{
					{CodeName: "alpha", Title: "Senior maintainability specialist", Guidance: "Coordinate architecture and code-quality findings into a phased execution roadmap."},
					{CodeName: "bravo", Title: "Staff software architect", Guidance: "Lead decomposition strategy and identify safe migration seams for large refactors."},
					{CodeName: "charlie", Title: "Testing strategy specialist", Guidance: "Close testability gaps and define automated guardrails for sustained maintainability."},
				},
			},
		},
	},
	{
		ID:          "xbrowser",
		Name:        "Cross-browser Issues",
		BeadPrefix:  "xbr",
		Description: "Surface browser-specific regressions and ensure consistent behavior.",
		FocusAreas: []string{
			"css feature compatibility",
			"javascript api support differences",
			"layout and rendering engine quirks",
			"input and event model variance",
			"polyfill and fallback strategy",
			"visual regression across browsers",
		},
		RoleConfigs: []AgentConfigRoles{
			{
				AgentCount: 1,
				Roles: []RoleDefinition{
					{CodeName: "alpha", Title: "Senior compatibility specialist", Guidance: "Own browser matrix triage and deliver fixes that preserve behavior across target platforms."},
				},
			},
			{
				AgentCount: 2,
				Roles: []RoleDefinition{
					{CodeName: "alpha", Title: "Senior compatibility specialist", Guidance: "Prioritize breakages by user share and define target support policy."},
					{CodeName: "bravo", Title: "Staff frontend platform engineer", Guidance: "Reproduce engine-specific bugs, implement cross-browser fixes, and validate parity."},
				},
			},
			{
				AgentCount: 3,
				Roles: []RoleDefinition{
					{CodeName: "alpha", Title: "Senior compatibility specialist", Guidance: "Coordinate browser coverage and convert findings into actionable implementation batches."},
					{CodeName: "bravo", Title: "Staff frontend platform engineer", Guidance: "Lead technical remediation and maintain compatibility test fixtures."},
					{CodeName: "charlie", Title: "Design systems specialist", Guidance: "Validate component-level visual and interaction consistency across engines."},
				},
			},
		},
	},
	{
		ID:          "a11y",
		Name:        "Accessibility Audit",
		BeadPrefix:  "a11y",
		Description: "Improve inclusive usability and compliance with accessibility standards.",
		FocusAreas: []string{
			"wcag success criteria coverage",
			"keyboard navigation and focus flow",
			"screen reader semantics and labels",
			"color contrast and visual cues",
			"forms and error feedback accessibility",
			"accessible component interaction patterns",
		},
		RoleConfigs: []AgentConfigRoles{
			{
				AgentCount: 1,
				Roles: []RoleDefinition{
					{CodeName: "alpha", Title: "Senior accessibility specialist", Guidance: "Own standards-based review and prioritize changes that unblock assistive technology users."},
				},
			},
			{
				AgentCount: 2,
				Roles: []RoleDefinition{
					{CodeName: "alpha", Title: "Senior accessibility specialist", Guidance: "Set WCAG priority matrix and guide remediation design decisions."},
					{CodeName: "bravo", Title: "Staff inclusive UX engineer", Guidance: "Implement semantic fixes and validate journeys with keyboard and screen reader testing."},
				},
			},
			{
				AgentCount: 3,
				Roles: []RoleDefinition{
					{CodeName: "alpha", Title: "Senior accessibility specialist", Guidance: "Coordinate audit scope and publish high-confidence remediation priorities."},
					{CodeName: "bravo", Title: "Staff inclusive UX engineer", Guidance: "Deliver implementation guidance and verify behavior in assistive tooling."},
					{CodeName: "charlie", Title: "Content and language specialist", Guidance: "Assess copy, instructions, and messaging clarity for cognitive accessibility."},
				},
			},
		},
	},
	{
		ID:          "errhandling",
		Name:        "Error Handling Audit",
		BeadPrefix:  "err",
		Description: "Strengthen resilience by improving failure detection and recovery paths.",
		FocusAreas: []string{
			"unhandled exceptions and panic boundaries",
			"error boundary and fallback UX",
			"retry logic and backoff policies",
			"timeout handling and cancellation",
			"observability and diagnostic context",
			"failure mode and effect analysis",
		},
		RoleConfigs: []AgentConfigRoles{
			{
				AgentCount: 1,
				Roles: []RoleDefinition{
					{CodeName: "alpha", Title: "Senior reliability specialist", Guidance: "Own end-to-end resilience review and define standards for robust failure handling."},
				},
			},
			{
				AgentCount: 2,
				Roles: []RoleDefinition{
					{CodeName: "alpha", Title: "Senior reliability specialist", Guidance: "Prioritize critical failure paths and set acceptable recovery expectations."},
					{CodeName: "bravo", Title: "Staff platform reliability engineer", Guidance: "Review boundary handling, retries, and telemetry to improve diagnosis and containment."},
				},
			},
			{
				AgentCount: 3,
				Roles: []RoleDefinition{
					{CodeName: "alpha", Title: "Senior reliability specialist", Guidance: "Coordinate resilience strategy and sequence improvements for high-risk flows first."},
					{CodeName: "bravo", Title: "Staff platform reliability engineer", Guidance: "Define robust timeout, retry, and fallback implementations with clear ownership."},
					{CodeName: "charlie", Title: "Incident response specialist", Guidance: "Improve observability and runbook readiness for rapid issue isolation and recovery."},
				},
			},
		},
	},
}
