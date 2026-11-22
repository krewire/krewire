# Workload: Declarative Infrastructure (`infra`)

The `infra` workload manages cloud infrastructure, provisioning blueprints, deployment targets, and environment states declaratively without requiring external DSLs like Terraform or HCL.

---

## 1. Key Capabilities

- **Declarative Go Blueprints (`pkg/cloud/infra/schema`)**: Strongly-typed Go definitions describing resources (containers, databases, object buckets, network ingress, routing).
- **Drift Detection & State Reconciliation (`pkg/cloud/infra/state`)**: Tracks desired state versus live cloud state, producing deterministic execution plans.
- **Plan Inspection (`krew build --plan`)**: Generates and inspects an execution graph before provisioning changes.
- **Provider Modularity**: Pluggable provider drivers supporting local Docker, Linux `systemd`, and cloud PaaS engines.

---

## 2. Directory Layout

The `krew new my-infra --infra` or `krew init --infra` scaffold writes:

```text
my-infra/
├── krewire.yaml
├── infra/
│   └── blueprint.go      # Declarative resource topology
└── README.md
```

---

## 3. Configuration (`krewire.yaml`)

```yaml
version: "0.1.0"
project:
  name: my-infra

infra:
  provider: "krewire-cloud"
  region: "asia-southeast1"
  state: ".krewire/infra/state.json"
  resources:
    - kind: "service"
      id: "api-backend"
      properties:
        image: "krewire/api:v0.2.0"
        replicas: 2
        port: 8080
    - kind: "database"
      id: "primary-db"
      properties:
        engine: "postgres"
        version: "16"
```

---

## 4. Implementation Example

### `infra/blueprint.go`
```go
package infra

type Resource struct {
	ID         string            `json:"id"`
	Kind       string            `json:"kind"`
	Properties map[string]string `json:"properties"`
}

type Blueprint struct {
	Name      string     `json:"name"`
	Region    string     `json:"region"`
	Resources []Resource `json:"resources"`
}

func GetBlueprint() Blueprint {
	return Blueprint{
		Name:   "production-cluster",
		Region: "asia-southeast1",
		Resources: []Resource{
			{
				ID:   "web-frontend",
				Kind: "container",
				Properties: map[string]string{
					"image": "krewire/web:latest",
					"port":  "3000",
				},
			},
			{
				ID:   "cache-store",
				Kind: "redis",
				Properties: map[string]string{
					"memory": "512mb",
				},
			},
		},
	}
}
```

---

## 5. Development & Deployment Commands

```bash
# Evaluate infrastructure blueprint and output plan to .krewire/build/plan.json
krew build --plan

# Preview staging changes without mutating cloud resources
krew deploy --plan
krew deploy --dry-run

# Apply and provision infrastructure
krew deploy
```
