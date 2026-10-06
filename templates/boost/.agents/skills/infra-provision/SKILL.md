---
name: infra-provision
description: Use for Krewire cloud infrastructure — provider abstraction, state/locking, plan/apply, AWS and Kubernetes, krewire deploy. Triggers: "infra", "deploy", "provider", "AWS", "Kubernetes", "terraform", "plan", "preview env".
---

# Infra Provision — Multi-Cloud Library-First IaC

Provision and deploy from `krewire.yaml`/`infra:` declarations. This skill covers `framework/infra` (spec `KWF-B7N3D`).

## 1. Detect the context

- Read `docs/project-vision.md` and `docs/specs/framework/KWF-INFRA-B7N3D-cloud-provider-abstraction.md`.
- Identify provider (`aws`/`k8s`), state backend (local `.krewire/state.json` vs S3/GCS + DynamoDB/Consul locking), and resources needed (`Compute`, `Database`, `Storage`, `Network`, `DNS`, `Certificate`, `SecretRef`).

## 2. Provider contract

- `Provider` interface: `Name()`, `Resources()`, `Create`/`Read`/`Update`/`Delete`, `Plan(desired) (Plan,error)`.
- `Plan` is pure, returns ordered `Action{Create,Update,Delete,NoOp}` topologically sorted by `DependsOn`.
- `Resource{Kind, ID, Properties, DependsOn}` round-trips through JSON state.

## 3. Schema & validation

- Go structs with `validate:"required"` via `libs/validation`; common kinds share canonical schema; provider-specific fields under `provider:` namespace.
- Secrets are `secret.Ref` (`env:` or secrets manager ARN) — never literal; state stores only identifiers.

## 4. State, locking, plan/apply

- Local: `.krewire/state.json`; Remote: S3/GCS with locking (DynamoDB/Consul, TTL, `ErrAlreadyLocked{Owner}`).
- `Plan` is side-effect-free; `Apply` persists after each resource in dependency order; repeating `Plan` with no change yields zero mutating actions.

## 5. Providers

- **AWS:** `Compute` → ECS/Fargate (default) or Lambda; `Database` → RDS/DynamoDB; `Storage` → S3 + CloudFront; `Network` → VPC/Subnet/SG/LB; `DNS` → Route53; `Certificate` → ACM; `SecretRef` → Secrets Manager.
- **Kubernetes:** manifests via `client-go` or `kubectl` fallback; kinds `Deployment`/`Service`/`Ingress`/`ConfigMap`/`Secret`/`HPA`; `Namespace` ordering enforced.

## 6. CLI flow

- `krewire build --plan` or `krewire deploy --plan` (dry-run) → review → `krewire deploy` → `krewire deploy --destroy`.
- `krewire deploy --preview` creates `pr-<number>` stacks annotated `krewire.io/preview` for GC.

## 7. Gates

- Fakes/`envtest` only in `go test ./...`; never live accounts in gates.
- Verify `krewire deploy --plan` prints ordered plan without mutating state; test idempotence and lock contention.

## Rules

Read `rules/key-rules.md` before acting.
