# Infra Provision — Key Rules

- Library-first: providers are importable; CLI is a thin wrapper over `infra.Plan`/`infra.Apply`.
- Plan is pure and side-effect-free; Apply is dependency-ordered and persists per resource.
- Secrets never land in state as values — only identifiers; literal secrets fail validation.
- Remote state must lock before mutation; stale locks expire via TTL; `ErrAlreadyLocked` is surfaced.
- Idempotence: repeating `Plan` with no desired change yields zero `Create`/`Update`/`Delete`.
- Preview envs are namespaced `pr-<number>` with `krewire.io/preview` annotation for GC.
- Tests use fakes/`envtest`; live AWS/K8s never required for gates.
