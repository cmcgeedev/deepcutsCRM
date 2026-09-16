# Deep Cuts: AWS cost rundown (before any Terraform)

Status: estimate, 2026-09-15. Prices are us-east-1 on-demand list prices as published in September 2026; all figures monthly, USD, 730 h/month. Chris's existing AWS account is past the 12-month free tier, so nothing below assumes free-tier credits.

## What the deployment is

Same lean shape as Conjurate: one Graviton (arm64) EC2 instance runs the single `deepcuts` binary behind Caddy (automatic TLS), SQLite on the instance's EBS volume, Litestream replicating the database to S3 continuously, proof photos synced to the same bucket nightly, secrets in SSM Parameter Store, deploys over SSM from GitHub Actions with OIDC. No load balancer, no RDS, no NAT gateway, no container registry (the binary is built in CI and copied to the box).

## Line items, one production environment

| Item | Sizing | Unit price | Monthly |
|---|---|---|---|
| EC2 t4g.small (2 vCPU, 2 GB) | 730 h | $0.0168/h | $12.26 |
| Public IPv4 address | 730 h | $0.005/h | $3.65 |
| EBS gp3 root + data | 20 GB | $0.08/GB | $1.60 |
| S3 Standard (Litestream replica, WAL segments, proofs) | ~3 GB | $0.023/GB | $0.07 |
| S3 requests (Litestream PUTs, nightly sync) | ~300k PUT | $0.005/1k | $1.50 |
| Route 53 hosted zone | 1 | $0.50 | $0.50 |
| Domain (.com via Route 53) | 1/yr | ~$13/yr | $1.08 |
| Data transfer out | < 100 GB (free tier) | $0 | $0.00 |
| CloudWatch alarms (CPU, disk, health) | 3 | first 10 free | $0.00 |
| SSM Parameter Store, SNS email, GitHub OIDC | standard tier | free | $0.00 |
| **Total, production only** | | | **≈ $20.66** |

Sensitivity:
- t4g.micro (1 GB) instead of small: saves $6.13, total ≈ $14.50. Go + SQLite + Caddy idle under 150 MB, so it works, but leaves no headroom for the phase-2 QuickBooks sync and for `sqlite3` maintenance on a populated database. Not recommended for production.
- t4g.medium (4 GB): adds $12.26, total ≈ $33. Only if phase 3 (lots/inventory) grows the working set well past a few hundred MB.
- 1-year no-upfront reserved instance or compute savings plan on t4g.small: ≈ $7.70 instead of $12.26, total ≈ $16. Worth it once the anchor customer is live and the box is known to stay.
- Litestream S3 requests are the only line that moves with usage. Sync interval 10 s (the default is 1 s) cuts PUTs roughly 10× at the cost of up to 10 s of data loss on an instance failure; at this volume that trade is fine. Budget $0.20 to $1.50.

## Nonprod

Same shape doubles the bill (≈ $41/month for two environments). Cheaper choices, in order of preference:
1. **No nonprod box.** `make demo` locally plus the CI e2e run is the staging story for phase 1; the app is a single binary with an embedded UI and a file database, so "prod-like" is a laptop. Cost: $0.
2. **Nonprod on t4g.micro, stopped when idle.** A stopped instance bills only its EBS ($0.80 for 10 GB) and nothing for the IPv4 while stopped only if the address is released (an allocated Elastic IP on a stopped instance still bills $3.65). Cost when used a few days a month: ≈ $2 to $4.
3. Full nonprod mirror: ≈ $20.

Recommendation: option 1 now, option 2 when phase 2's QuickBooks sandbox needs a URL Intuit can reach.

## One-time and near-zero items

- Domain registration: ~$13/year (included above as $1.08/month).
- Terraform state bucket + DynamoDB lock table: cents.
- GitHub Actions minutes: the free plan's 2,000 minutes/month covers the ~3-minute CI run many times over.
- mkcert/TLS: none; Caddy obtains Let's Encrypt certificates for free.

## Alternatives considered

- **Lightsail** (2 GB bundle, $12/month, IPv4 and 3 TB transfer included): ≈ $14 all-in with S3 and DNS. Roughly $6 cheaper than EC2 on-demand and about the same as a reserved t4g.small, but it gives up SSM deploys, IAM/OIDC from CI, and the Terraform patterns Conjurate already has. Not worth diverging for $6.
- **Serverless (Lambda + Aurora/DSQL or DynamoDB)**: would require abandoning SQLite and the single-connection design. Not a cost decision at this scale; ruled out by the spec.
- **Two products on one box**: Deep Cuts and Conjurate could share a t4g.small (two binaries, two Caddy sites, two SQLite files). Saves ≈ $17/month but couples two customers' uptime and deploys. Keep separate for now.

## Bottom line

| Scenario | Monthly |
|---|---|
| Production only, on-demand | ≈ $21 |
| Production only, 1-yr reserved | ≈ $16 |
| Production + stopped-when-idle nonprod | ≈ $23 to $25 |
| Production + full nonprod mirror | ≈ $41 |

Recommendation: production on a t4g.small on-demand (≈ $21/month), no nonprod box, reserve the instance after the anchor customer has been live for a month. That is the Conjurate shape at the Conjurate price, and every line above the instance itself is under $4.

Sources: EC2 on-demand pricing (aws.amazon.com/ec2/pricing/on-demand, instances.vantage.sh), EBS pricing (aws.amazon.com/ebs/pricing), public IPv4 charge (aws.amazon.com/vpc/pricing), S3 pricing (cloudzero.com/blog/s3-pricing), Route 53 pricing (aws.amazon.com/route53/pricing), Lightsail pricing (cloudburn.io/blog/amazon-lightsail-pricing).
