# Tasks

## TODO

- [x] Phase 0: Establish the strict-host prerequisite contract from code and existing admin APIs <!-- t:prereq -->
- [x] Phase 0: Create a dedicated deployment ticket, design guide, and diary <!-- t:ticket -->
- [x] Phase 1: Rename the local-only server command to `serve-dev` without a compatibility alias <!-- t:rename -->
- [x] Phase 1: Update all maintained CLI reference pages from `serve` to `serve-dev` <!-- t:docs-rename -->
- [x] Phase 2: Write a repeatable local provisioning script for SQLite, keys, user, client, token secret, and TLS <!-- t:provision -->
- [x] Phase 2: Write a foreground strict-host launcher and health/discovery probe script <!-- t:launch -->
- [x] Phase 3: Add a systemd hardening template and operator runbook <!-- t:systemd -->
- [ ] Phase 3: Add a container deployment reference after deciding the supported TLS-termination topology <!-- t:container -->
- [x] Phase 4: Run the real admin provisioning sequence and browser device-verification smoke <!-- t:real-smoke -->
- [x] Phase 4: Diagnose and fix the post-approval strict-host unresponsiveness before treating device authorization as release-ready <!-- t:blocker -->
- [ ] Phase 5: Add automated strict-host browser/device smoke coverage with a locally trusted certificate <!-- t:automated-smoke -->
- [ ] Phase 5: Complete backup/restore drill, external review, and production release checklist <!-- t:release -->
