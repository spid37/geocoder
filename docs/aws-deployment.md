# AWS deployment plan

Plan for hosting the GNAF geocoder on AWS with an externally built, read-only SQLite database (~6 GB).

## Goals

- Keep the current architecture: Go API + local SQLite (no Postgres/D1 migration)
- Ship the database as a versioned artifact (built outside the serve host)
- Simple ops close to the current Dokploy setup
- Optional Cloudflare only for DNS / TLS / WAF in front

## Target architecture

```text
[DB build machine / CI]
        │  upload versioned artifact
        ▼
   S3 (gnaf/gnaf-YYYY-MM.db + latest.json)
        │  pull on deploy / refresh
        ▼
   EC2 (ap-southeast-2)
     • Docker: geocoder serve --db /data/gnaf.db
     • EBS gp3 ~30 GB (DB + headroom)
     • optional Cloudflare DNS/WAF in front
        │
        ▼
   clients → :443 → (Cloudflare or ALB) → :8080
```

**Why EC2 first (not ECS/Fargate):** simplest for a single sticky 6 GB disk, cheapest at low–medium traffic, closest to Dokploy. Move to ECS later if multi-instance orchestration is needed.

## Components

| Piece | Choice | Notes |
| --- | --- | --- |
| Region | `ap-southeast-2` | Closest for AU users / G-NAF |
| Compute | 1× EC2 (`t3.medium` or `m6i.large` if RAM helps cache) | ~4–8 GB RAM; SQLite benefits from page cache |
| Disk | EBS gp3 30 GB | 6 GB DB + headroom; do not rely on a tiny root volume alone |
| Artifact store | Private S3 bucket | Versioned keys; lifecycle policy for old DBs |
| Image | Existing Dockerfile → ECR (or GHCR) | No app rewrite |
| Secrets | SSM Parameter Store / Secrets Manager | `API_KEY` |
| Health | `/health` (+ optional `/health/data`) | Origin / load balancer checks |
| TLS | Cloudflare proxy **or** ALB + ACM | Cloudflare-only is fine if already in use |
| Backups | S3 is source of truth; EBS snapshots optional | Disk can be rebuilt from S3 anytime |

## Data and deploy flow

### DB publish (once per G-NAF release)

1. Build `gnaf.db` outside the serve host (local, CI, or a dedicated build box).
2. Upload to `s3://<bucket>/gnaf/gnaf-YYYY-MM.db`.
3. Update `s3://<bucket>/gnaf/latest.json`:

```json
{
  "key": "gnaf/gnaf-YYYY-MM.db",
  "sha256": "<hex>"
}
```

### App deploy

1. Pull the container image from ECR (or GHCR).
2. Sync the DB from S3 only when `latest.json` (or object etag/version) has changed — avoid downloading 6 GB on every restart.
3. Start the container with `/data` on the EBS volume.
4. Serve read-only (`geocoder serve` already opens the DB in read-only mode).

### Cold start tip

Prefer a persistent EBS volume (or a baked AMI) so routine restarts do not re-download the full database.

## Phased rollout

### Phase 0 — Decisions (locked defaults)

| Topic | Default |
| --- | --- |
| HA | Single instance; brief deploy downtime is OK |
| Edge | Cloudflare DNS / TLS / WAF in front of EC2 |
| IaC | Manual console + scripts for v1; Terraform later if useful |
| DB build | External to the serve host |

### Phase 1 — Skeleton

- VPC (or default VPC) and security group: `80`/`443` from Cloudflare (or public), `8080` only from trusted front door if applicable
- EC2 + 30 GB EBS
- IAM instance role: S3 read for the DB bucket, ECR pull
- S3 bucket + first DB upload
- Bootstrap script: sync DB → run Docker
- Smoke test `/health` and `/v1/geocode`

### Phase 2 — Production hardening

- `API_KEY` from SSM (or Secrets Manager)
- Cloudflare DNS pointing at the origin
- Deploy path: build image → push ECR → restart via SSM/SSH
- DB refresh runbook: upload artifact → update `latest.json` → sync → restart
- Alarms: CPU, disk, instance status, optional 5xx if metrics are available

### Phase 3 — Optional later

- Second instance + ALB (each replica keeps a **local copy** of the DB)
- AMI baking so scale-out does not wait on a full S3 download
- Autoscaling only if QPS justifies it

## Out of scope

- Migrating to RDS / Postgres
- Cloudflare D1 (or other edge SQLite) as the primary store
- Running `setup` / G-NAF download on the serve host
- Sharing one SQLite file over EFS/NFS

## Docker serve command (unchanged)

```bash
geocoder serve --db /data/gnaf.db --data-dir /data --port 8080
```

Mount `/data` from the EBS volume. Set `API_KEY` from SSM at runtime when auth is required.

## Open follow-ups

- Confirm instance size after measuring real QPS and RSS with the 6 GB DB loaded
- Choose ECR vs GHCR for the image registry
- Write the sync/bootstrap and DB publish scripts when implementation starts
