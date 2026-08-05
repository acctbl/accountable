# 0001: Environment foundations and the first development cell

- Status: accepted
- Date: 2026-08-05

## Context

Accountable needs real AWS and Infisical adapter proofs without turning staging into a breakable experiment account.
It also needs a clean promotion boundary and must not pay for cells before they are needed.

## Decision

Keep separate OpenTofu roots and state keys for development, staging, production, and security/backup.
Each root pins its AWS account ID and fails closed when the authenticated account differs.
Do not use OpenTofu workspaces as environment boundaries.

Manage linked-account budgets and CloudTrail delegation from a separate organization root in the AWS Organizations management account.
Keep organization audit evidence in the security/backup account behind a private, versioned, KMS-encrypted bucket that workload accounts cannot administer.
The organization trail records validated management events across all accounts and Regions.

The first complete cell is in development.
Its edge is CloudFront with AWS WAF and Shield Standard, a private encrypted web bucket, and `/api/*` routed through a VPC origin to an internal ALB and private ECS Fargate tasks.
The database is private, encrypted, Multi-AZ PostgreSQL.
Cell data uses a dedicated encrypted S3 bucket and KMS key.
API, migration, bootstrap, and ECS execution permissions stay separate.
Runtime images are selected by digest.
Infisical owns application and migration secrets; AWS Secrets Manager owns only the RDS-managed master secret used by bootstrap.

The cell has one lifecycle setting:

- `ephemeral` is restricted to development create-prove-destroy cells.
  RDS keeps one day of backups and no final snapshot, the ALB has deletion protection disabled, and S3 may remove all object versions during destroy.
- `durable` is required for staging and production.
  RDS keeps 35 days of point-in-time recovery, automated backups, deletion protection, and a final snapshot; the ALB has deletion protection enabled; and S3 must not erase objects to make destroy succeed.

The initial proof edge uses the generated `*.cloudfront.net` hostname and CloudFront default certificate.
AWS requires all other viewer-certificate fields to be omitted in that configuration and fixes its security policy to `TLSv1`.
Enforcing TLS 1.2 or newer therefore requires a later custom hostname and an ACM certificate in `us-east-1`; the default-certificate configuration must not claim a stronger policy because AWS would normalize it and leave a perpetual plan difference.

Development cells follow this cycle: create from empty state, bootstrap, migrate, preflight, deploy, smoke test, prove the next plan is empty, retain only for the proof window, then destroy.
A lasting staging cell is created only when the team deliberately chooses to retain it.

## Consequences

Durable cell destruction is intentionally two-step and reviewable: protections must first be explicitly disabled in a reviewed apply.
A custom domain and certificate are prerequisites before any lasting public environment can enforce TLS 1.2 or newer at CloudFront.
Production signing, production promotion, full observability correlation, autoscaling, RDS Proxy, and multi-cell routing remain outside this milestone.
