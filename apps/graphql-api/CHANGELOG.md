# Changelog

## [0.0.3](https://github.com/rahulmathews/project-neo/compare/graphql-api-v0.0.2...graphql-api-v0.0.3) (2026-05-08)


### Features

* **graphql-api:** production hardening ([ac1727c](https://github.com/rahulmathews/project-neo/commit/ac1727c4268b8b00b4e089dd5473524e9cc3a34e))
* **graphql-api:** production hardening ([edd0f8b](https://github.com/rahulmathews/project-neo/commit/edd0f8b0bc080c55f0281c6cd46251abebf57f36))

## [0.0.2](https://github.com/rahulmathews/project-neo/compare/graphql-api-v0.0.1...graphql-api-v0.0.2) (2026-05-07)


### Bug Fixes

* **graphql-api:** keep read-only lookup queries unauthenticated ([8a0ceaa](https://github.com/rahulmathews/project-neo/commit/8a0ceaa57e76374051609672d7650fae785413db))
* **graphql-api:** update websocket init callback ([c1d17fc](https://github.com/rahulmathews/project-neo/commit/c1d17fce8f904971c985983dabeef403ad07a801))
* harden auth flows and correct message insert signaling ([90bac3e](https://github.com/rahulmathews/project-neo/commit/90bac3e701567f046c9942c2d3b05682cf0691e7))
* harden graphql auth checks and message insert dedupe signaling ([66c4a2e](https://github.com/rahulmathews/project-neo/commit/66c4a2efd20b54f3d87789b8cf75856d3df8d3b4))
* **shared:** harden auth and message dedupe handling ([d2a7dfe](https://github.com/rahulmathews/project-neo/commit/d2a7dfef81607fe93213dfe566c2fb1940e1d59e))
* tighten auth checks and dedupe insert signaling ([32dded2](https://github.com/rahulmathews/project-neo/commit/32dded286c83242830482f6417076b5e514e4e13))

## 0.0.1 (2026-04-19)


### Features

* **graphql-api:** add domain models and regenerate gqlgen ([f7d0265](https://github.com/rahulmathews/project-neo/commit/f7d0265e9aab62f38584ee0880bc12c50fd5da61))
* **graphql-api:** add postgres implementations for user, group, location ([41707c8](https://github.com/rahulmathews/project-neo/commit/41707c8a4fac9941cc27f829f47466a9f52f3244))
* **graphql-api:** add PostgreSQL LISTEN/NOTIFY listener ([5f2ec30](https://github.com/rahulmathews/project-neo/commit/5f2ec3099099cac027555b82e60e9e7b66a97398))
* **graphql-api:** add repository interfaces ([66a41cc](https://github.com/rahulmathews/project-neo/commit/66a41ccde076add7f34875e2c52fc3d43a16038d))
* **graphql-api:** add Supabase JWT auth middleware ([8f502fa](https://github.com/rahulmathews/project-neo/commit/8f502fad792e98ebda5ba8a79521dcc9d813ed1c))
* **graphql-api:** implement mutation resolvers ([3a1c1f7](https://github.com/rahulmathews/project-neo/commit/3a1c1f75e1be410d106b99baae8aab0bd04117e9))
* **graphql-api:** implement query and field resolvers ([0958524](https://github.com/rahulmathews/project-neo/commit/09585248f2128a47249c8f0acd00289101fb0260))
* **graphql-api:** implement subscription resolvers ([80cb355](https://github.com/rahulmathews/project-neo/commit/80cb355886be7f5185ad270fd6d4769c807b0f11))
* **graphql-api:** split schema into files and update gqlgen config ([e40b362](https://github.com/rahulmathews/project-neo/commit/e40b3629c9936d4b25be06021f449a8d3b2f8c12))
* **graphql-api:** wire repositories and auth middleware in main ([b29e683](https://github.com/rahulmathews/project-neo/commit/b29e683956fc0230a75bc108c48d4d46f53de137))


### Bug Fixes

* Moved whatsmeow states to out of scope of supabase ([6ffcaa1](https://github.com/rahulmathews/project-neo/commit/6ffcaa1c7f6608dcc656aef477e5d9fb4fcb4a47))
