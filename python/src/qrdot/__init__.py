"""Official Python client for the QR. API."""

from .client import Qrdot, QrdotApiError, verify_webhook_signature

__all__ = ["Qrdot", "QrdotApiError", "verify_webhook_signature"]
__version__ = "0.1.1"
