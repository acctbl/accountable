---
tool: trivy-iac
finding: AVD-AWS-0011
kind: misconfig_id
expires: 2026-11-01
owner: "@acctbl/maintainers"
reason: The web edge distribution serves only public immutable release assets and the public runtime config document from a private bucket, so managed WAF rules add recurring cost without a protected surface yet.
residual_risk: Volumetric and bot traffic against the web edge is mitigated only by CloudFront defaults until a WAF is attached.
---

The CloudFront distribution in infra/opentofu/web.tf has no aws_wafv2_web_acl association.
Attach one before the web app exposes an authenticated surface or launch traffic warrants managed rules.
