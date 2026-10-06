# Philosophy — Krewire Guild

## Philosophy

**AI-native constitution for the unified framework.** One install gives any Krewire project (any of 8 kinds) a set of agents/skills that already know the vision, the 8-kind detection, the full `kiw` matrix, and  specs.

**Principles:**

- **Template is source of truth.** `template/` is what `guild.go` embeds; `guild/` repo docs govern the module, `template/AGENTS.md` governs projects that install it.
- **Vision-aware.** Agents load `https://github.com/krewire/internal/blob/main/docs/project-vision.md` compact first, then the relevant `KWF-*` spec.
- **Spec per initiative.** `KWG-*` specs live in this repo (`guild/docs/specs/`).


## Contribution

- Read [`project-vision.md`](https://github.com/krewire/internal/blob/main/docs/project-vision.md) and `docs/specs/index.md` before changing behavior.
- Add/update tests matching project patterns; keep suite green.
- Update `README.md` / `docs/` and specs when public behavior changes; follow ecosystem spec conventions.
