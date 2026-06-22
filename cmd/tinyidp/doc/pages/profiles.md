---
Title: "Profiles (profiles.yaml)"
Slug: profiles
Short: "Named bundles of OIDC config overrides selected with --profile, sitting above defaults and below config/env/flags."
Topics:
- profiles
- config
- oidc
Commands:
- serve
Flags:
- profile
- profile-file
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

Profiles are named bundles of field overrides stored in a YAML file. They let
you switch an entire OIDC provider setup with one flag, without retyping
flags or maintaining separate config files per environment.

A typical use is environment presets: a `dev` profile that points at a local
issuer and a `ci` profile that uses a different client and callback.

## File format

The file is a YAML map. The top level is the profile name, the second level is
the section slug, and the third level is field name/value pairs.

```yaml
dev:
  oidc:
    client-id: dev-profile-client
    addr: 127.0.0.1:6600

ci:
  oidc:
    client-id: ci-runner
    addr: 127.0.0.1:6601
    redirect-uris:
      - http://localhost:9090/callback
```

Every field available under `oidc:` (see `tinyidp help oidc-config`) can be set
in a profile.

## Default file location

`tinyidp serve` looks for a default profile file at:

```
~/.config/tinyidp/profiles.yaml
```

(resolved from `os.UserConfigDir`, so `$XDG_CONFIG_HOME/tinyidp/profiles.yaml`
on Linux). If this default file is missing and the requested profile is
`default`, profile loading is skipped silently — `tinyidp serve` works out of
the box with no profiles.yaml.

## Selecting a profile

```
tinyidp serve --profile dev
tinyidp serve --profile ci --profile-file /path/to/profiles.yaml
```

Or via environment variables:

```
TINYIDP_PROFILE=dev tinyidp serve
TINYIDP_PROFILE_FILE=/path/to/profiles.yaml tinyidp serve
```

## Precedence

From lowest to highest:

1. Section defaults
2. **Profiles** (this layer)
3. Config files (`--config-file`)
4. Environment variables (`TINYIDP_*`)
5. Positional arguments
6. CLI flags

So a profile overrides the built-in defaults, but a config file, an env var,
or a flag overrides the profile. This makes a profile a convenient baseline
that local overrides always win against.

## Error behavior

- If `--profile-file` points at a file that does not exist **and** it is not
  the default file, `tinyidp serve` errors out. A typo in `--profile-file`
  never silently runs with defaults.
- If the default file is missing and the requested profile is `default`,
  loading is skipped silently (the no-profiles.yaml case).
- If the default file is missing and a non-default profile is requested,
  `tinyidp serve` errors (e.g. `TINYIDP_PROFILE=staging` with no
  profiles.yaml).

## Introspection

To see which profile was applied and which source won each field:

```
tinyidp serve --profile dev --print-parsed-fields
```

The parse log includes `source: profiles` entries for every field the profile
overrode.

## See also

- `tinyidp help oidc-config` — the fields available under the `oidc` section.
- `tinyidp help config-files` — how `--config-file` layers above profiles.
