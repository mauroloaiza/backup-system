"""
BackupSMC — Server Entry Point
"""
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

# TODO: importar routers cuando estén implementados
# from app.api.v1 import auth, nodes, jobs, backups, destinations

app = FastAPI(
    title="BackupSMC API",
    description="API REST del sistema de backup empresarial BackupSMC",
    version="0.1.0",
    docs_url="/docs",
    redoc_url="/redoc",
)

# CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # TODO: leer de settings en producción
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.get("/health", tags=["health"])
async def health_check():
    """Health check endpoint."""
    return {"status": "ok", "service": "backupsmc-server"}


# TODO: registrar routers
# app.include_router(auth.router, prefix="/api/v1/auth", tags=["auth"])
# app.include_router(nodes.router, prefix="/api/v1/nodes", tags=["nodes"])
# app.include_router(jobs.router, prefix="/api/v1/jobs", tags=["jobs"])
