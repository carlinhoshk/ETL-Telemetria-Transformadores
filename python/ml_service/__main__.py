"""Entry point so `python -m ml_service` runs the HTTP service."""
from .service import main

if __name__ == "__main__":
    raise SystemExit(main())
