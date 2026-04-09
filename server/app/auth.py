"""
Authentication — JWT for web users, API token for agents.
"""
from datetime import datetime, timedelta, timezone
from typing import Optional

from fastapi import Depends, HTTPException, Header, status
from fastapi.security import OAuth2PasswordBearer
from jose import JWTError, jwt
from passlib.context import CryptContext
from sqlalchemy.orm import Session
import os

from app.database import get_db

SECRET_KEY = os.getenv("SECRET_KEY", "backupsmc-change-me-in-production-32ch")
ALGORITHM = "HS256"
ACCESS_TOKEN_EXPIRE_MINUTES = 60 * 8  # 8 hours

# Token agents must send in X-Agent-Token header
AGENT_TOKEN = os.getenv("AGENT_TOKEN", "backupsmc-agent-secret")

pwd_context = CryptContext(schemes=["sha256_crypt"], deprecated="auto")
oauth2_scheme = OAuth2PasswordBearer(tokenUrl="/api/v1/auth/login")


def verify_password(plain: str, hashed: str) -> bool:
    return pwd_context.verify(plain, hashed)


def hash_password(password: str) -> str:
    return pwd_context.hash(password)


def create_access_token(data: dict, expires_delta: Optional[timedelta] = None) -> str:
    to_encode = data.copy()
    expire = datetime.now(timezone.utc) + (
        expires_delta or timedelta(minutes=ACCESS_TOKEN_EXPIRE_MINUTES)
    )
    to_encode["exp"] = expire
    return jwt.encode(to_encode, SECRET_KEY, algorithm=ALGORITHM)


def get_current_user(
    token: str = Depends(oauth2_scheme),
    db: Session = Depends(get_db),
):
    """Dependency for JWT-protected web endpoints."""
    from app.models import User  # local import to avoid circular

    exc = HTTPException(
        status_code=status.HTTP_401_UNAUTHORIZED,
        detail="Credenciales inválidas",
        headers={"WWW-Authenticate": "Bearer"},
    )
    try:
        payload = jwt.decode(token, SECRET_KEY, algorithms=[ALGORITHM])
        username: str = payload.get("sub")
        if not username:
            raise exc
    except JWTError:
        raise exc

    user = db.query(User).filter(User.username == username).first()
    if not user or not user.is_active:
        raise exc
    return user


def require_agent_token(
    x_agent_token: str = Header(None),
    db: Session = Depends(get_db),
):
    """Dependency for agent-only endpoints — validates against agent_tokens table."""
    from app.models import AgentToken
    from datetime import datetime, timezone

    if not x_agent_token:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="Token requerido")

    token = db.query(AgentToken).filter(
        AgentToken.token == x_agent_token,
        AgentToken.is_active == True,  # noqa: E712
    ).first()

    if not token:
        # backward compat: also accept the env-var token
        if x_agent_token != AGENT_TOKEN:
            raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="Token de agente inválido")
    else:
        token.last_used_at = datetime.now(timezone.utc)
        db.commit()
