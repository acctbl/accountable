# Structure

```text
accountable/
├── apps/
│   ├── web/                 product SPA
│   └── ops/                 ops (placeholder)
├── cmd/
│   ├── api/                 API server
│   ├── migrate/             discovers and applies owner migrations
│   ├── preflight/           startup checks, no listen
│   └── webconfig/           renders the validated web runtime config deploy artifact
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
│   └── opentofu/            private S3 + CloudFront web edge
├── scripts/                 local helpers
├── tests/                   Playwright
├── testdata/                negative fixtures
└── docs/                    docs
```

```text
browser → CloudFront edge → private S3 (apps/web release + cmd/webconfig artifact)
browser → apps/web → cmd/api → internal/modules/*
                              → internal/bootstrap → internal/platform/*
```
