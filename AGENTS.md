# AGENTS.md

## Scope
These instructions apply to the whole `GoOne` repository. Prefer code over docs when they disagree; `doc/framwork.md` explicitly notes some README/history drift.

## Big picture
- `GoOne` is a Go microservice game backend built around `Application -> router -> bus -> TransactionMgr`, with newer IDL-driven `ssrpc` layered on top of the same SSPacket/transaction runtime.
- Main services live under `src/`: `connsvr` (client TCP/WS gateway), `mainsvr` (player logic), `infosvr` (brief profile/cache), `mysqlsvr` (persistence), `roomcentersvr` (room lifecycle), `web_svr` (Gin HTTP admin APIs).
- Typical internal flow is: client packet -> `connsvr` -> `lib/service/router` -> bus/routing by `BusId` and `module/misc.ServerRouteRules` -> target service `globals.TransMgr`.
- Every service follows `lib/service/application/app.go`: implement `OnInit/OnReload/OnProc/OnTick/OnExit`, call `application.Init(...)`, then `application.Run()`.

## Runtime/config conventions
- Shared config shape is defined in `common/gconf/config.go`; services read the `-svr_conf` flag and unmarshal their section plus `base_cfg`.
- Sample local configs are `common/gconf/server_conf_ide.yaml` and `env/server_conf_ide.yaml` (if both exist in your branch, inspect the one your startup path uses).
- `BusId` embeds service type; routing rules are centralized in `module/misc/constant.go`. Do not invent per-service routing logic without checking `svrinstmgr` expectations.
- `mainsvr` and `roomcentersvr` run `TransMgr.InitAndRun(..., true, ...)` for serialized per-router work; `connsvr`/`infosvr` do not. Preserve that concurrency model when adding handlers.

## Handler patterns you should follow
- Legacy command handlers are still common: register them before `TransMgr.InitAndRun()` using `globals.TransMgr.RegisterCmd(...)` from each service's `cmd_handler/register.go`.
- `ssrpc` is the newer path. Generated registration lives under `api/gen/**`; services usually register legacy handlers first, then generated `ssrpc` handlers so wrappers can override migrated commands (see `src/mainsvr/app.go`, `src/roomcentersvr/app.go`).
- When a handler requires loaded domain state, use the repo’s adapters instead of duplicating fetch logic: `src/mainsvr/cmd_handler/role_adapter.go` and `src/roomcentersvr/cmd_handler/adapter.go`.
- Web APIs are different: `src/web_svr/app.go` boots Gin via `lib/web/web_gin` and mounts routes through `controller.LoadWebRoutes`, not through the bus-driven transaction loop.

## Generated code / safe edit boundaries
- Do **not** hand-edit generated files: `api/gen/**`, `game_protocol/protocol/*.pb.go`, `common/gamedata/repository/**/*.gen.go`.
- Game data repositories self-register in `init()` and hot-swap with `atomic.Value` (example: `common/gamedata/repository/global_config/GlobalData.gen.go`); change the source data/tooling, not the generated repository code.
- `go.mod` uses `replace github.com/Iori372552686/game_protocol => ./game_protocol`, so protocol changes belong in the local `game_protocol/` module and must be regenerated consistently.

## Workflows that matter here
- Preferred top-level entrypoint is `main.sh`; start with:
  - `./main.sh doctor`
  - `./main.sh build`
  - `./main.sh build web`
- `build.sh` still contains legacy targets for services no longer present under `src/`; do not assume it reflects the full current topology. For services missing there (for example `roomcentersvr`), build directly with `go build -o build/roomcentersvr ./src/roomcentersvr`.
- Local dependencies come from `env/env_docker.yaml` (MySQL/Redis/ZooKeeper/RabbitMQ). Some tests are integration-like and expect those services running.
- On Windows, prefer PowerShell for proto generation (`.\scripts\proto_goone.ps1`) and WSL/Git-Bash for `main.sh`/`build.sh`.

## Integration points to inspect before deep changes
- Registry/config/bus backends are chosen by URI-like config strings in YAML (`register_addr`, `bus_mq_addr`) and implemented under `lib/contrib/**` and `lib/service/bus/**`.
- `common/gamedata/gamedata.go` supports local file loading and Nacos hot reload; services may initialize either path depending on `NacosConf`.
- Deployment/ops behavior is in `deploy/README.md`, `deploy/deploy.sh`, and `deploy/scripts/server.sh`; environment management is in `env/README.md` and `env/docker.sh`.

