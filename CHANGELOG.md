# Change Log

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

## [0.4.0] - 2022-12-16

### Added

- Limitador version as env var by @eguzki in [#37](https://github.com/Kuadrant/limitador-operator/pull/37)
- Automate CSV generation by @didierofrivia in [#44](https://github.com/Kuadrant/limitador-operator/pull/44)
- Redis storage for counters by @didierofrivia in [#48](https://github.com/Kuadrant/limitador-operator/pull/48)
- Smart replica reconciliation by @eguzki in [#52](https://github.com/Kuadrant/limitador-operator/pull/52)
- Building multi platform images by @didierofrivia in [#55](https://github.com/Kuadrant/limitador-operator/pull/55)

### Changed

- Limitador configuration using command line args instead of env vars by @eguzki in [#47](https://github.com/Kuadrant/limitador-operator/pull/47)
- Golang v1.18 and k8s API v0.24.2 by @eguzki in [#50](https://github.com/Kuadrant/limitador-operator/pull/50)

### Fixed

- Configmap reconciliation by @eguzki in [#42](https://github.com/Kuadrant/limitador-operator/pull/42)
- Controller resources by @eguzki in [#41](https://github.com/Kuadrant/limitador-operator/pull/41)
- Service name by @eguzki in [#43](https://github.com/Kuadrant/limitador-operator/pull/43)
- Error handling and status reconciliation by @eguzki in [#51](https://github.com/Kuadrant/limitador-operator/pull/51)
- Controller owner references instead of owner references to watch for changes by @eguzki in [#53](https://github.com/Kuadrant/limitador-operator/pull/53)
- Support invalid limits configmap by @eguzki in [#54](https://github.com/Kuadrant/limitador-operator/pull/54)

## [0.3.0] - 2022-08-11

### Changed

- Create LICENSE by @thomasmaas in [#12](https://github.com/Kuadrant/limitador-operator/pull/12)
- initial version of the README by @eguzki in [#13](https://github.com/Kuadrant/limitador-operator/pull/13)
- Enhanced logging by @eguzki in [#14](https://github.com/Kuadrant/limitador-operator/pull/14)
- SDK/K8s/Go Version updates and bundle generation by @mikenairn in [#15](https://github.com/Kuadrant/limitador-operator/pull/15)
- manifest files for easy prototyping by @rahulanand16nov in [#17](https://github.com/Kuadrant/limitador-operator/pull/17)
- Add image build GH action by @mikenairn in [#18](https://github.com/Kuadrant/limitador-operator/pull/18)
- Updating tooling by @didierofrivia in [#23](https://github.com/Kuadrant/limitador-operator/pull/23)
- Kubebuilder-tools workaround for darwin/arm64 arch by @didierofrivia in [#25](https://github.com/Kuadrant/limitador-operator/pull/25)
- Limitador Service Settings by @didierofrivia in [#19](https://github.com/Kuadrant/limitador-operator/pull/19)
- Reconciling limits `ConfigMap` by @didierofrivia in [#28](https://github.com/Kuadrant/limitador-operator/pull/28)
- Limits in limitador by @didierofrivia in [#26](https://github.com/Kuadrant/limitador-operator/pull/26)
- Support for empty Limtador CR by @eguzki in [#30](https://github.com/Kuadrant/limitador-operator/pull/30)
- [**BREAKING CHANGE**] remove RateLimit CRD leftovers by @eguzki in [#32](https://github.com/Kuadrant/limitador-operator/pull/32)
- remove kube-rbac-proxy sidecar by @eguzki in [#31](https://github.com/Kuadrant/limitador-operator/pull/31)
- Updating status by @didierofrivia in [#33](https://github.com/Kuadrant/limitador-operator/pull/33)
- Enhance status reconciling cycle by @didierofrivia in [#34](https://github.com/Kuadrant/limitador-operator/pull/34)

## New Contributors
- @thomasmaas made their first contribution in [#12](https://github.com/Kuadrant/limitador-operator/pull/12)
- @mikenairn made their first contribution in [#15](https://github.com/Kuadrant/limitador-operator/pull/15)
- @rahulanand16nov made their first contribution in [#17](https://github.com/Kuadrant/limitador-operator/pull/17)
- @didierofrivia made their first contribution in [#23](https://github.com/Kuadrant/limitador-operator/pull/23)

## [0.2.0] - 2021-09-28

### Changed

- Fix internal links from 3scale to kuadrant [#11](https://github.com/Kuadrant/limitador-operator/pull/11)

## [0.1.1] - 2021-08-16

### Changed

- Leverage ownerreferences for cleanup on CR removal [#9](https://github.com/Kuadrant/limitador-operator/pull/9)

## [0.1.0] - 2021-07-15

### Added

- Initial release
- Limitador CRD [#2](https://github.com/Kuadrant/limitador-operator/pull/2)
- RateLimit CRD [#6](https://github.com/Kuadrant/limitador-operator/pull/6)

[Unreleased]: https://github.com/Kuadrant/limitador-operator/compare/compare/v0.4.0...HEAD
[0.3.0]: https://github.com/Kuadrant/limitador-operator/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/Kuadrant/limitador-operator/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/Kuadrant/limitador-operator/compare/v0.1.0...v0.1.1
