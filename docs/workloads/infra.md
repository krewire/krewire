# Workload: Declarative Infrastructure (`infra`)

The `infra` workload manages cloud infrastructure, provisioning blueprints, deployment targets, and environment states declaratively.

---

## 1. Key Capabilities

- **Declarative Blueprints (`packages/cloud/infra/schema`)**: Typed Go schemas describing cloud resources (compute, containers, object buckets, networking).
- **State & Provider Management (`packages/cloud/infra/state`)**: Track infrastructure changes, detect drift, and generate deterministic deployment plans.
- **Unified Deployment (`packages/cloud/deploy`)**: Zero-downtime rolling deploys, canary stages, and artifact distribution into cloud runners.

---

## 2. Directory Layout

```text
my-infra/
├── krewire.yaml          # Project configuration (kind: infra)
├── infra/                # Infrastructure definitions
│   ├── production.go
│   └── staging.go
└── state/                # Environment state lock files
```

---

## 3. Configuration (`krewire.yaml`)

```yaml
version: "0.1.0"
project:
  name: my-infra
  kind: infra

infra:
  provider: "krewire-cloud"
  region: "asia-southeast1"
```

---

## 4. Commands

```bash
# Evaluate infrastructure blueprint and output plan
kiw build --plan

# Deploy or stage infrastructure
kiw deploy --preview
kiw deploy
```
