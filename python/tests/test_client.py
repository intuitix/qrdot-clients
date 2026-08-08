import hashlib
import hmac
import io
import json
import time
from unittest.mock import MagicMock, patch
from urllib.error import HTTPError

import pytest

from qrdot import Qrdot, QrdotApiError, verify_webhook_signature


def _http_response(body: bytes, status: int = 200, content_type: str = "application/json"):
    cm = MagicMock()
    cm.__enter__.return_value = cm
    cm.__exit__.return_value = False
    cm.status = status
    cm.read.return_value = body
    cm.headers = {"Content-Type": content_type}
    return cm


def test_rejects_bad_api_key():
    with pytest.raises(ValueError):
        Qrdot("bad")


def test_create_qr():
    client = Qrdot("sk_test_abc")
    payload = {"id": "qr_1", "shortCode": "abc", "targetUrl": "https://example.com"}
    with patch(
        "urllib.request.urlopen",
        return_value=_http_response(json.dumps(payload).encode()),
    ):
        out = client.qr.create({"targetUrl": "https://example.com"})
    assert out["id"] == "qr_1"


def test_api_error():
    client = Qrdot("sk_test_abc")
    body = json.dumps(
        {"error": {"code": "validation", "message": "bad"}}
    ).encode()
    http_err = HTTPError(
        "https://api.qrdot.dev/v1/qr",
        400,
        "Bad Request",
        hdrs=None,
        fp=io.BytesIO(body),
    )
    with patch("urllib.request.urlopen", side_effect=http_err):
        with pytest.raises(QrdotApiError) as ei:
            client.qr.create({"targetUrl": "nope"})
    assert ei.value.code == "validation"
    assert ei.value.status == 400


def test_image_bytes():
    client = Qrdot("sk_test_abc")
    with patch(
        "urllib.request.urlopen",
        return_value=_http_response(b"\x89PNG", content_type="image/png"),
    ):
        data, mime = client.qr.image("qr_1", "png")
    assert data.startswith(b"\x89PNG")
    assert mime == "image/png"


def test_verify_webhook_signature():
    secret = "whsec_test"
    body = '{"ok":true}'
    t = int(time.time())
    sig = hmac.new(
        secret.encode(), f"{t}.{body}".encode(), hashlib.sha256
    ).hexdigest()
    header = f"t={t},v1={sig}"
    assert verify_webhook_signature(secret, body, header) is True
    assert verify_webhook_signature(secret, body + "x", header) is False


def test_domains_and_webhook_parity_paths():
    client = Qrdot("sk_test_abc")
    assert hasattr(client, "domains")
    seen: list[str] = []

    def fake_urlopen(req, timeout=60):
        seen.append(f"{req.get_method()} {req.full_url.removeprefix(client.base_url)}")
        return _http_response(b'{"ok":true}')

    with patch("urllib.request.urlopen", side_effect=fake_urlopen):
        client.domains.create({"hostname": "go.example.com"})
        client.domains.list()
        client.domains.get("dom_1")
        client.domains.dns("dom_1")
        client.domains.dns_provider("dom_1")
        client.domains.domain_connect_start("dom_1")
        client.domains.forward_dns("dom_1", {"email": "a@b.com"})
        client.domains.refresh("dom_1")
        client.domains.set_default("dom_1")
        client.domains.delete("dom_1")
        client.webhooks.list_deliveries("wh_1", limit=10)
        client.webhooks.replay(
            "wh_1", {"qr_id": "qr_1", "scan_id": "sc_1", "ts": "2026-01-01T00:00:00Z"}
        )
        client.assets.presign_logo("image/png", filename="logo.png")
        client.assets.get_logo("logo_1")
        client.assets.complete_logo("logo_1", filename="logo.png")
        client.create_qr({"targetUrl": "https://example.com"})

    assert "POST /v1/domains" in seen
    assert "GET /v1/domains" in seen
    assert "GET /v1/domains/dom_1/dns-provider" in seen
    assert "POST /v1/domains/dom_1/domain-connect/start" in seen
    assert "GET /v1/webhooks/wh_1/deliveries?limit=10" in seen
    assert "POST /v1/webhooks/wh_1/replay" in seen
    assert "POST /v1/assets/logo/presign" in seen
    assert "GET /v1/assets/logo/logo_1" in seen
    assert "POST /v1/qr" in seen
