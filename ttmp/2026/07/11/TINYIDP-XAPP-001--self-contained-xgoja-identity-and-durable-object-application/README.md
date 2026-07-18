# Self-Contained xgoja Identity and Durable Object Application

This is the document workspace for ticket TINYIDP-XAPP-001.

Start with the [design guide](./design-doc/01-self-contained-xgoja-tiny-idp-express-and-durable-objects-analysis-design-and-implementation-guide.md), then use [tasks.md](./tasks.md) and the [implementation diary](./reference/01-investigation-diary.md) to resume at the first unchecked stable task ID.

## Structure

- **design-doc/**: Intern-ready analysis, architecture, and implementation guide
- **reference/**: Reference documentation and API contracts
- **playbooks/**: Operational playbooks and procedures
- **scripts/**: Utility scripts and automation
- **sources/**: External sources and imported documents
- **various/**: Scratch or meeting notes, working notes
- **archive/**: Optional space for deprecated or reference-only artifacts

## Implementation repositories

- `tiny-idp`: product ticket, embedded identity provider, and eventual custom host.
- `../go-go-goja`: OIDC relying party, app sessions, planned Express routes, generated runtime package, and host services.
- `../go-go-objects`: actor-bound persistent JavaScript objects and xgoja provider.

## Getting Started

Use docmgr commands to manage this workspace:

- Add documents: `docmgr doc add --ticket TINYIDP-XAPP-001 --doc-type design-doc --title "My Design"`
- Import sources: `docmgr import file --ticket TINYIDP-XAPP-001 --file /path/to/doc.md`
- Update metadata: `docmgr meta update --ticket TINYIDP-XAPP-001 --field Status --value review`
