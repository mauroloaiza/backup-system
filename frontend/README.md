# BackupSMC — Frontend (React)

Dashboard web para gestión de backups. Construido con React 18 + Vite + Tailwind CSS.

## Stack

- React 18 + TypeScript
- Vite 5 (bundler)
- Tailwind CSS v3
- shadcn/ui (componentes)
- Recharts (gráficas)
- React Query (estado servidor)
- React Router v6
- Zustand (estado global)

## Páginas

| Ruta | Descripción |
|------|-------------|
| `/login` | Inicio de sesión |
| `/dashboard` | Resumen general y estado de jobs |
| `/nodes` | Gestión de nodos/agentes |
| `/jobs` | Jobs de backup (CRUD) |
| `/history` | Historial de ejecuciones |
| `/destinations` | Configurar destinos de almacenamiento |
| `/settings` | Configuración y usuarios |

## Desarrollo

```bash
npm install
npm run dev
```

Ver [DEVELOPMENT.md](../docs/DEVELOPMENT.md) para más detalles.
