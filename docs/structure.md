# Structure

## Repository Structure

```text
accountable/
├── apps/
│   ├── web/               product SPA
│   └── ops/               ops (placeholder)
├── cmd/
│   ├── api/               API server
│   ├── web/               serves built SPA
│   ├── migrate/           applies migrations
│   └── preflight/         startup checks, no listen
├── internal/
│   ├── foundation/        secrets, DB, storage, crypto
│   ├── server/            Connect handlers + interceptors
│   ├── appconfig/         API config
│   ├── apierror/          problem details
│   ├── migration/         goose runner
│   ├── platform/          shared primitives
│   └── configfile/        TOML helpers
├── proto/                 wire contracts
├── gen/go/                generated Go (buf)
├── packages/
│   ├── proto/             generated TS (buf)
│   └── ui/                shared React UI
├── db/                    SQL migrations
├── config/                environment TOML
├── infra/                 OpenTofu
├── scripts/               local helpers
├── tests/                 Playwright
├── testdata/              negative fixtures
└── docs/                  docs
```

```text
browser → apps/web → cmd/api → internal/server
                              → internal/foundation → postgres / secrets / storage / crypto
```

## Glossary
