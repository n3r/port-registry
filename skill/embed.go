package skill

import _ "embed"

//go:embed port-manager/SKILL.md
var SkillMD []byte

//go:embed port-manager/references/WORKFLOW.md
var WorkflowMD []byte
