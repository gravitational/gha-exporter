# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.0.10] - 2024-03-19
### Added
- Added workflow elapsed time metric

### Changed
- Renamed `gha_workflow_run_time_seconds` to `gha_workflow_run_runner_seconds`

## [0.0.9] - 2024-03-18
### Fixed
- Fixed ref labels on PRs using PR head branch instead of base

## [0.0.8] - 2024-03-16
### Added
- Added event type, and workflow reference based on event type

## [0.0.7] - 2024-03-08
### Fixed
- Switched container image base to distroless

## [0.0.6] - 2024-03-08
### Fixed
- Fixed Helm chart using wrong image name

## [0.0.5] - 2024-03-08
### Fixed
- Fixed incorrect secret manifest formatting

## [0.0.4] - 2024-03-07
### Added
- Added a Helm chart

### Fixed
- Fixed container image being deployed to `ghcr.io/gravitational/gha-exporter/gha-exporter` instead of `ghcr.io/gravitational/gha-exporter`

## [0.0.3] - 2024-03-07
### Fixed
- AMD64 container images not starting due to binary being dynamically linked

## [0.0.2] - 2024-03-07
### Added
- Initial release. GHA exporter provides Prometheus metrics for Github Action runs.

[Unreleased]: https://github.com/gravitational/gha-exporter/compare/v0.0.10...HEAD
[0.0.10]: https://github.com/gravitational/gha-exporter/compare/v0.0.9...v0.0.10
[0.0.9]: https://github.com/gravitational/gha-exporter/compare/v0.0.8...v0.0.9
[0.0.8]: https://github.com/gravitational/gha-exporter/compare/v0.0.7...v0.0.8
[0.0.7]: https://github.com/gravitational/gha-exporter/compare/v0.0.6...v0.0.7
[0.0.6]: https://github.com/gravitational/gha-exporter/compare/v0.0.5...v0.0.6
[0.0.5]: https://github.com/gravitational/gha-exporter/compare/v0.0.4...v0.0.5
[0.0.4]: https://github.com/gravitational/gha-exporter/compare/v0.0.3...v0.0.4
[0.0.3]: https://github.com/gravitational/gha-exporter/compare/v0.0.2...v0.0.3
[0.0.2]: httpx://github.com/gravitational/gha-exporter/releases/tag/v0.0.2
