# CHANGELOG

<a name="v0.11.2"></a>
## [v0.11.2] - 2022-04-26
### Bug Fixes
- **plugin:** allow specifying sub directories of remote modules as src

<a name="v0.11.1"></a>
## [v0.11.1] - 2022-04-18
### Bug Fixes
- print error if fail to open plugin
- **doc:** setup field was deprecated

<a name="v0.11.0"></a>
## [v0.11.0] - 2022-04-15
### Bug Fixes
- **plugin:** fix issue with plugin build failure in Go1.18

### Features
- enable to marshal scenarios into YAML
- **mock:** enable to assert request
- **template:** allow writing left arrow function call in map syntax
- **template:** enable to use template in map keys
- **template:** enable to escape { by \

<a name="v0.10.0"></a>
## [v0.10.0] - 2022-01-31
### Bug Fixes
- update the go directive of go.mod
- **plugin:** use the same module version as scenarigo for building plugins

### BREAKING CHANGE

This package requires Go 1.17 or later.

<a name="v0.9.0"></a>
## [v0.9.0] - 2021-12-03
### Bug Fixes
- **errors:** Errors returns nil if no errors

### Code Refactoring
- use yaml.PathBuilder to specify the pos

### Features
- add plugin sub-command
- add setup feature
- add "scenarigo plugin list" command
- add "scenarigo config validate" command
- **plugin:** enable to build plugin from remote "go gettable" src
- **plugin:** enable registration of setup functions to be executed for each scenario
- **template:** add bool literals

<a name="v0.8.1"></a>
## [v0.8.1] - 2021-09-27
### Bug Fixes
- add workaround to avoid the bug of Go 1.17

### Code Refactoring
- export functions

### Features
- list command refers to the configuration file
- remove blank lines from logs

### BREAKING CHANGE

"file" and "verbose" options are removed from the list sub-command.

<a name="v0.8.0"></a>
## [v0.8.0] - 2021-09-08
### Bug Fixes
- enable CGO on release build
- **query:** do not extract by the inline field name
- **template:** fix a bug by nil struct field
- **template:** marshal variables to YAML in LAF arguments
- **template:** keep the original memory address
- **template:** marshal LAF arguments with indent

### Features
- enable cross compile with CGO
- **grpc:** loose type checking for equaler
- **template:** execute templates of data
- **version:** get version from build info

<a name="v0.7.0"></a>
## [v0.7.0] - 2021-07-30
### Bug Fixes
- **assert:** fix the assertion operators
- **assert:** fix the logic to compare Go protobuf APIv2 messages
- **grpc:** rename body field to message
- **query:** don't access unexported field

### Code Refactoring
- don't use ioutil package

### Features
- change default configuration filename
- enable to set configurations by file
- add WithConfig option
- colorize outputs
- support NO_COLOR standard
- enable strictly check on request field
- use Go protobuf APIv2
- **assert:** enable to change the behavior of equal assertion
- **query:** allow accessing anonymous fields

### Performance Improvements
- reuse parsed AST node to print error tokens

### BREAKING CHANGE

This package requires Go 1.16 or later.

<a name="v0.6.3"></a>
## [v0.6.3] - 2021-04-08
### Bug Fixes
- enable to bind vars defined in the included scenario

<a name="v0.6.2"></a>
## [v0.6.2] - 2021-04-07
### Bug Fixes
- **plugin:** avoid the error caused by loading plugins concurrently ([#78](https://github.com/zoncoen/scenarigo/issues/78))

### Code Refactoring
- **assert:** remove query from arguments

### Features
- **assert:** add length assertion
- **assert:** add greaterThan/greaterThanOrEqual/lessThan/lessThanOrEqual ([#77](https://github.com/zoncoen/scenarigo/issues/77))
- **reporter:** enable to generate test report ([#83](https://github.com/zoncoen/scenarigo/issues/83))
- **reporter:** include the execution time of sub-tests ([#82](https://github.com/zoncoen/scenarigo/issues/82))

<a name="v0.6.1"></a>
## [v0.6.1] - 2021-01-14
### Bug Fixes
- **template:** don't convert invalid values to avoid panic

<a name="v0.6.0"></a>
## [v0.6.0] - 2021-01-12
### Bug Fixes
- **template:** enable to set to pointer values

### Features
- export RunScenario function
- add WithScenariosFromReader option
- allow template in header assertion
- **assert:** add regexp function
- **context:** add ScenarioFilePath

<a name="v0.5.1"></a>
## [v0.5.1] - 2020-10-23
### Bug Fixes
- **template:** restore funcs in args of left arrow function

### Features
- **assert:** add "and" function

<a name="v0.5.0"></a>
## [v0.5.0] - 2020-10-05
### Features
- **assert:** add "or" function
- **expect:** enable strict option when decoding yaml for expect to prevent field misplacement ([#59](https://github.com/zoncoen/scenarigo/issues/59))
- **grpc:** allow using a template as code and msg
- **http:** allow using a template as code

<a name="v0.4.0"></a>
## [v0.4.0] - 2020-09-02
### Bug Fixes
- register errdetails proto messages to unmarshal Any
- **expect:** use the default assertion if no expect ([#55](https://github.com/zoncoen/scenarigo/issues/55))
- **template:** avoid to panic ([#54](https://github.com/zoncoen/scenarigo/issues/54))

### Features
- **cmd:** add list sub-command ([#51](https://github.com/zoncoen/scenarigo/issues/51))

<a name="v0.3.3"></a>
## [v0.3.3] - 2020-06-17
### Bug Fixes
- **core:** add generated files to avoid the import error ([#41](https://github.com/zoncoen/scenarigo/issues/41))
- **deps:** update YAML library ( v1.7.12 => v1.7.15 ) ([#47](https://github.com/zoncoen/scenarigo/issues/47))
- **deps:** update YAML library ( v1.7.10 => v1.7.11 ) ([#42](https://github.com/zoncoen/scenarigo/issues/42))
- **deps:** update YAML library to fix a bug ( v1.7.9 => v1.7.10 ) ([#40](https://github.com/zoncoen/scenarigo/issues/40))
- **template:** fix processing for variadic arguments of function ([#48](https://github.com/zoncoen/scenarigo/issues/48))

<a name="v0.3.2"></a>
## [v0.3.2] - 2020-06-15
### Bug Fixes
- **deps:** update YAML library to fix a bug ( v1.7.8 => v1.7.9 ) ([#39](https://github.com/zoncoen/scenarigo/issues/39))

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


[v0.11.2]: https://github.com/zoncoen/scenarigo/compare/v0.11.1...v0.11.2
[v0.11.1]: https://github.com/zoncoen/scenarigo/compare/v0.11.0...v0.11.1
[v0.11.0]: https://github.com/zoncoen/scenarigo/compare/v0.10.0...v0.11.0
[v0.10.0]: https://github.com/zoncoen/scenarigo/compare/v0.9.0...v0.10.0
[v0.9.0]: https://github.com/zoncoen/scenarigo/compare/v0.8.1...v0.9.0
[v0.8.1]: https://github.com/zoncoen/scenarigo/compare/v0.8.0...v0.8.1
[v0.8.0]: https://github.com/zoncoen/scenarigo/compare/v0.7.0...v0.8.0
[v0.7.0]: https://github.com/zoncoen/scenarigo/compare/v0.6.3...v0.7.0
[v0.6.3]: https://github.com/zoncoen/scenarigo/compare/v0.6.2...v0.6.3
[v0.6.2]: https://github.com/zoncoen/scenarigo/compare/v0.6.1...v0.6.2
[v0.6.1]: https://github.com/zoncoen/scenarigo/compare/v0.6.0...v0.6.1
[v0.6.0]: https://github.com/zoncoen/scenarigo/compare/v0.5.1...v0.6.0
[v0.5.1]: https://github.com/zoncoen/scenarigo/compare/v0.5.0...v0.5.1
[v0.5.0]: https://github.com/zoncoen/scenarigo/compare/v0.4.0...v0.5.0
[v0.4.0]: https://github.com/zoncoen/scenarigo/compare/v0.3.3...v0.4.0
[v0.3.3]: https://github.com/zoncoen/scenarigo/compare/v0.3.2...v0.3.3
[v0.3.2]: https://github.com/zoncoen/scenarigo/compare/v0.3.1...v0.3.2
[v0.3.1]: https://github.com/zoncoen/scenarigo/compare/v0.3.0...v0.3.1
[v0.3.0]: https://github.com/zoncoen/scenarigo/compare/v0.2.0...v0.3.0
[v0.2.0]: https://github.com/zoncoen/scenarigo/compare/v0.1.0...v0.2.0
