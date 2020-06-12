# CHANGELOG

<a name="v0.3.1"></a>
## [v0.3.1] - 2020-06-12
### Bug Fixes
- **core:** fix ctx.Response() for http protocol ([#35](https://github.com/zoncoen/scenarigo/issues/35))
- **errors:** fix incorrect line number in YAML source ([#38](https://github.com/zoncoen/scenarigo/issues/38))

<a name="v0.3.0"></a>
## [v0.3.0] - 2020-06-11
### Features
- **core:** support to output error with YAML ([#33](https://github.com/zoncoen/scenarigo/issues/33))

<a name="v0.2.0"></a>
## [v0.2.0] - 2020-06-03
### Code Refactoring
- **core:** replace YAML libraries to goccy/go-yaml ([#31](https://github.com/zoncoen/scenarigo/issues/31))

### Features
- **core:** read YAML files only as scenarios ([#28](https://github.com/zoncoen/scenarigo/issues/28))
- **grpc:** enable to check header/trailer metadata of gRPC response ([#29](https://github.com/zoncoen/scenarigo/issues/29))
- **http:** enable to check HTTP response headers ([#30](https://github.com/zoncoen/scenarigo/issues/30))

### BREAKING CHANGE

change protocl.Protocol interface

<a name="v0.1.0"></a>
## v0.1.0 - 2020-05-17
- first release


[v0.3.1]: https://github.com/zoncoen/scenarigo/compare/v0.3.0...v0.3.1
[v0.3.0]: https://github.com/zoncoen/scenarigo/compare/v0.2.0...v0.3.0
[v0.2.0]: https://github.com/zoncoen/scenarigo/compare/v0.1.0...v0.2.0
