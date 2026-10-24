# Workload: Declarative Infrastructure (`infra`)

The `infra` workload manages cloud infrastructure, provisioning blueprints, deployment targets, and environment states declaratively.

---

## 1. Key Capabilities

- **Declarative Blueprints (`pkg/cloud/infra/schema`)**: Typed Go schemas describing cloud resources (compute, containers, object buckets, networking).
- **State & Provider Management (`pkg/cloud/infra/state`, `pkg/cloud/infra/provider`)**: Track infrastructure changes, detect drift, and generate deterministic deployment plans.
- **Unified Deployment (`pkg/cloud/deploy`)**: Rolling deploys and artifact distribution into cloud runners.

---

## 2. Directory Layout

The `kiw init --infra` scaffold writes:

```text
my-infra/
├── krewire.yaml
├── infra/
│   └── blueprint.go      # Declarative resource definitions
└── README.md
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
  state: ".krewire/infra/state.json"
  resources:
    - kind: "service"
      id: "api"
      properties:
        image: "krewire/api:latest"
```

---

## 4. Commands

```bash
# Evaluate infrastructure blueprint and output a plan
kiw build --plan

# Deploy or stage infrastructure
kiw deploy --plan
kiw deploy
```
