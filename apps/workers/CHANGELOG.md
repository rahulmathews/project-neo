# Changelog

## 1.0.0 (2026-04-12)


### Features

* **docker:** add multi-stage Dockerfiles and docker-compose ([2d15ec3](https://github.com/rahulmathews/project-neo/commit/2d15ec3a07975e9aa5798fade5d07226889dc217))
* **graphql-api:** scaffold gqlgen with health query ([580a944](https://github.com/rahulmathews/project-neo/commit/580a9443990e460aadc91535582565edac840743))
* **workers:** add Claude Haiku fallback extractor ([3c68375](https://github.com/rahulmathews/project-neo/commit/3c683754756583a2b3f0341f13461048d6a85631))
* **workers:** add Connector interface, WhatsApp client, and message handler ([085caf3](https://github.com/rahulmathews/project-neo/commit/085caf3e11e137b9828b4dff161b9a61e5401735))
* **workers:** add internal MessageWriter and GroupSourceReader store layers ([9f51aef](https://github.com/rahulmathews/project-neo/commit/9f51aef706a80d939c2b67c1ae288840a8695ec8))
* **workers:** add location context resolver for parser ([3ec8bc4](https://github.com/rahulmathews/project-neo/commit/3ec8bc47019c97c56e0e96f235112e128bad0063))
* **workers:** add ParsedRide struct and ErrNotARide sentinel ([74cbdd8](https://github.com/rahulmathews/project-neo/commit/74cbdd84f3556541695b30467604abd018b38e83))
* **workers:** add parser extractor pipeline (regex → haiku → location → write) ([ba5d43e](https://github.com/rahulmathews/project-neo/commit/ba5d43ef5634f831e32485b160943c0089e0c284))
* **workers:** add parser writer — ride insert and message status updates ([8fc1012](https://github.com/rahulmathews/project-neo/commit/8fc1012d571e57aad9860bb4cf78035dc2cb5316))
* **workers:** add pg_notify listener for message parser ([04690b7](https://github.com/rahulmathews/project-neo/commit/04690b73f4b48899d07483ca445a4223314c25d2))
* **workers:** add regex-based ride message extractor ([303edd2](https://github.com/rahulmathews/project-neo/commit/303edd2d3f4e89a4d39a29c4dde3c3a72326b939))
* **workers:** connect-then-discover — remove pre-seeded sources requirement ([5ff2b1d](https://github.com/rahulmathews/project-neo/commit/5ff2b1da65f4030c718a0cc608faa23bd9e8c2d3))
* **workers:** extract seats_available from ride messages ([0a51544](https://github.com/rahulmathews/project-neo/commit/0a51544cfe38de9bd0b8dc6fcd0ad7e6c8808ade))
* **workers:** initialize Go modules and enable workspace ([080c783](https://github.com/rahulmathews/project-neo/commit/080c78333fe3be323ba581755e17bb9821442447))
* **workers:** wire up message parser listener in run.go ([856efac](https://github.com/rahulmathews/project-neo/commit/856efac688d9275fa1fa992cdca3bd05af1c3900))
* **workers:** wire WhatsApp connector into service entrypoint ([d67656b](https://github.com/rahulmathews/project-neo/commit/d67656b3ec8d6d5b2d1915261f27a40d54658e04))


### Bug Fixes

* **docker:** correct workers container port from 8081 to 8083 ([11dc077](https://github.com/rahulmathews/project-neo/commit/11dc07716477d21addd0c88a22d98457a2151b47))
* **workers:** call container.Upgrade to create whatsmeow schema on first run ([5571118](https://github.com/rahulmathews/project-neo/commit/5571118ca51e6a21b9dd7153b91daa2cc2567b67))
* **workers:** clarify raw SQL alias and extend date phrase to two-digit days ([8c132ee](https://github.com/rahulmathews/project-neo/commit/8c132ee807387e9e6e20d9d6b6c260c086135ce7))
* **workers:** connector cleanup, remove phantom number param, fix context threading ([260bf7c](https://github.com/rahulmathews/project-neo/commit/260bf7cf425b8e74393119397d2125a9125f9395))
* **workers:** fix departure time parsing — uppercase AM/PM and support dot separator ([be666bb](https://github.com/rahulmathews/project-neo/commit/be666bb626bb742e1ba975aeff92a7e656767a54))
* **workers:** make trigger idempotent and reuse Haiku client ([4e5209f](https://github.com/rahulmathews/project-neo/commit/4e5209ff5412191bcd966f6bfec11eaabf9a5ba7))
* **workers:** parse cost when currency symbol follows number (e.g. 10$) ([413ece5](https://github.com/rahulmathews/project-neo/commit/413ece5e24bf2762b7f71bf528b25f1eb78151f9))
* **workers:** remove unused table alias from duplicate check query and add composite index ([75bcfa3](https://github.com/rahulmathews/project-neo/commit/75bcfa3a16ce20c32d1c980f6384fb0b60927ec6))
* **workers:** render QR code as terminal block using qrterminal ([2527268](https://github.com/rahulmathews/project-neo/commit/252726814a6fff3f4a061eed7a229c4fc112634e))
* **workers:** resolve golangci-lint errors in whatsapp client and handler ([9344333](https://github.com/rahulmathews/project-neo/commit/9344333b646ad305c4dbd709d962d20c358f41eb))
* **workers:** set UUID on message before insert to prevent pkey collision ([542b56d](https://github.com/rahulmathews/project-neo/commit/542b56dfefa71fd0e1f8de095bdeb0bc7cc5f582))
* **workers:** skip duplicate ride creation for same content hash in group ([51517d2](https://github.com/rahulmathews/project-neo/commit/51517d23411163a3ae50081ab9d6d96456775288))
* **workers:** strip temporal, seat, and conversational suffixes from location text ([371c3ff](https://github.com/rahulmathews/project-neo/commit/371c3ffda771d29b1837275ce3f26fc315abb1a5))
* **workers:** strip trailing time/cost/distance from captured location text ([9e5659f](https://github.com/rahulmathews/project-neo/commit/9e5659f797b143a5e1b071115fd48fde5e8f93d2))
* **workers:** wire seats_available through to ride insert ([5fb089f](https://github.com/rahulmathews/project-neo/commit/5fb089f85b698f23ccb1799773c24d7c758658dc))
* **workers:** wrap NewConnectors error, inject logger into startHealthServer ([a5e4ea9](https://github.com/rahulmathews/project-neo/commit/a5e4ea9ebd8ee14dcfa676a37c60ebc10e025212))
