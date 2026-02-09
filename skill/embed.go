package skill

import _ "embed"

//go:embed port-registry/SKILL.md
var SkillMD []byte

//go:embed port-registry/references/WORKFLOW.md
var WorkflowMD []byte
