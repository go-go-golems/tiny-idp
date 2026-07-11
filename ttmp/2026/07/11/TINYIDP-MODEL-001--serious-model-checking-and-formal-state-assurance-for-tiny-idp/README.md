# Serious Model Checking and Formal State Assurance for tiny-idp

This is the long-lived research and implementation workspace for moving
tiny-idp from model-based testing and finite-history linearizability checks to a
serious bounded model-checking program. Start with `index.md`, then read the
primary design guide and diary before selecting a task.

## Structure

- **design/**: Design documents and architecture notes
- **reference/**: Reference documentation and API contracts
- **playbooks/**: Operational playbooks and procedures
- **scripts/**: Utility scripts and automation
- **sources/**: External sources and imported documents
- **various/**: Scratch or meeting notes, working notes
- **archive/**: Optional space for deprecated or reference-only artifacts

## Getting Started

Use docmgr commands to manage this workspace:

- Add documents: `docmgr doc add --ticket TINYIDP-MODEL-001 --doc-type design-doc --title "My Design"`
- Import sources: `docmgr import file --ticket TINYIDP-MODEL-001 --file /path/to/doc.md`
- Update metadata: `docmgr meta update --ticket TINYIDP-MODEL-001 --field Status --value review`
