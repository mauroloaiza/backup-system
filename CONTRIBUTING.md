# Guía de contribución — BackupSMC

¡Gracias por tu interés en contribuir a BackupSMC!
Este documento describe el proceso y las convenciones para contribuir al proyecto.

---

## Tabla de contenidos

1. [Código de conducta](#código-de-conducta)
2. [¿Cómo puedo contribuir?](#cómo-puedo-contribuir)
3. [Configuración del entorno de desarrollo](#configuración-del-entorno-de-desarrollo)
4. [Convenciones de código](#convenciones-de-código)
5. [Convenciones de commits](#convenciones-de-commits)
6. [Proceso de Pull Request](#proceso-de-pull-request)
7. [Branching strategy](#branching-strategy)

---

## Código de conducta

Este proyecto sigue el [Código de Conducta](./CODE_OF_CONDUCT.md). Al participar, se espera que lo respetes.

---

## ¿Cómo puedo contribuir?

### Reportar bugs
- Usa la plantilla de [Bug Report](./.github/ISSUE_TEMPLATE/bug_report.md)
- Incluye pasos para reproducirlo, comportamiento esperado y real, y logs si aplica

### Solicitar funcionalidades
- Usa la plantilla de [Feature Request](./.github/ISSUE_TEMPLATE/feature_request.md)
- Explica el problema que resuelve, no solo la solución propuesta

### Contribuir código
1. Abre un issue primero para discutir el cambio
2. Haz fork del repositorio
3. Crea una rama desde `develop` (ver [Branching strategy](#branching-strategy))
4. Implementa tu cambio
5. Abre un Pull Request siguiendo el template

---

## Configuración del entorno de desarrollo

Ver [DEVELOPMENT.md](./docs/DEVELOPMENT.md) para instrucciones detalladas.

---

## Convenciones de código

### Python (server/)
- Formateador: `black`
- Linter: `ruff`
- Type hints: obligatorios en funciones públicas
- Docstrings: estilo Google

```bash
cd server
black .
ruff check .
```

### Go (agent/)
- Formateador: `gofmt` / `goimports`
- Linter: `golangci-lint`

```bash
cd agent
gofmt -w .
golangci-lint run
```

### TypeScript/React (frontend/)
- Formateador: `prettier`
- Linter: `eslint`

```bash
cd frontend
npm run lint
npm run format
```

---

## Convenciones de commits

Usamos [Conventional Commits](https://www.conventionalcommits.org/):

```
<tipo>(<alcance>): <descripción corta>

[cuerpo opcional]

[footer opcional]
```

### Tipos permitidos

| Tipo | Cuándo usarlo |
|------|---------------|
| `feat` | Nueva funcionalidad |
| `fix` | Corrección de bug |
| `docs` | Solo documentación |
| `style` | Formato, sin cambio de lógica |
| `refactor` | Refactorización sin feat ni fix |
| `test` | Agregar o corregir tests |
| `chore` | Tareas de mantenimiento (deps, CI, etc.) |
| `perf` | Mejora de rendimiento |
| `ci` | Cambios en CI/CD |

### Ejemplos

```
feat(server): add S3 destination support
fix(agent): handle connection timeout gracefully
docs(readme): update quick start instructions
chore(deps): bump fastapi to 0.115.0
```

---

## Proceso de Pull Request

1. El PR debe apuntar a la rama `develop`
2. Completa el template de PR
3. Asegúrate de que todos los checks de CI pasen
4. Se requiere al menos **1 aprobación** antes de hacer merge
5. Usa **Squash and Merge** para mantener el historial limpio

---

## Branching strategy

Seguimos **Git Flow simplificado**:

```
main          → producción estable (tags de release)
develop       → integración continua
feature/*     → nuevas funcionalidades
fix/*         → correcciones de bugs
hotfix/*      → correcciones urgentes en producción
release/*     → preparación de release
```

### Reglas

- **Nunca** hacer push directo a `main` o `develop`
- Las ramas `feature/` y `fix/` salen de `develop`
- Las ramas `hotfix/` salen de `main`
- Usar nombres descriptivos: `feature/s3-destination`, `fix/agent-timeout`
