# Static Analysis and Implementation Verification for tiny-idp

This ticket designs and tracks the maintained static-analysis and selected
implementation-verification program for tiny-idp. Start with `index.md`, then
read the primary design guide and diary before changing the analyzer backlog.

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

- Add documents: `docmgr doc add --ticket TINYIDP-STATIC-001 --doc-type design-doc --title "My Design"`
- Import sources: `docmgr import file --ticket TINYIDP-STATIC-001 --file /path/to/doc.md`
- Update metadata: `docmgr meta update --ticket TINYIDP-STATIC-001 --field Status --value review`
