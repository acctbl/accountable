# Structure

```text
accountable/
├── apps/
│   ├── web/                 product SPA
│   └── ops/                 ops (placeholder)
├── cmd/
│   ├── api/                 API server
│   ├── migrate/             discovers and applies owner migrations
│   └── preflight/           startup checks, no listen
├── internal/
│   ├── bootstrap/           fail-closed dependency composition
│   ├── modules/
│   │   ├── system/          system API
│   │   └── probe/           non-prod architecture probe
│   ├── platform/
│   │   ├── clock/
│   │   ├── secret/
│   │   ├── database/        + Goose SQL
│   │   ├── storage/
│   │   ├── crypto/
│   │   ├── features/
│   │   └── awsconfig/
│   ├── server/              Connect interceptors
│   ├── appconfig/
│   ├── apierror/
│   ├── migration/           goose runner + catalogue
│   └── configfile/
├── proto/                   wire contracts
├── gen/go/                  generated Go (buf)
├── packages/
│   ├── proto/               generated TS (buf)
│   └── ui/                  shared React UI
├── config/                  environment TOML
├── infra/
│   └── opentofu/
├── scripts/                 local helpers
├── tests/                   Playwright
├── testdata/                negative fixtures
└── docs/                    docs
```

```text
browser → apps/web → cmd/api → internal/modules/*
                              → internal/bootstrap → internal/platform/*
```
