# Packaging Scaffold

This directory contains package manager specific templates and configuration for
surveiller release automation.

Current scope (Phase 1):
- Base nfpm config skeleton for `.deb` and `.rpm`
- Homebrew formula template
- Chocolatey nuspec template
- Makefile helper targets for package build flow

Planned follow-up phases:
- Release manifest generation and version normalization
- Package artifact build scripts
- Repository/tap/feed publication scripts
- CI smoke tests for install verification

## Usage Note

Use Makefile helper targets to bootstrap package build flow:

- `make package-help` to list all package helper targets
- `make package-manifest` to generate release manifest (Phase 2)
- `make package-linux` to build `.deb`/`.rpm` artifacts (US1)
- `make package-homebrew` to render Homebrew formula payload (US1)
- `make package-choco` to build Chocolatey payload (US1)
- `make package-build` to run all package helper steps

Some targets intentionally skip in Phase 1 until corresponding scripts are added
in later tasks.

## Layout

```
packaging/
├── choco/
│   └── surveiller.nuspec.tmpl
├── homebrew/
│   └── surveiller.rb.tmpl
├── nfpm/
│   └── nfpm.yaml
└── release-manifest.schema.json
```
