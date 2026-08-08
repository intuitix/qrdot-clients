from __future__ import annotations

import hashlib
import hmac
import json
import time
import urllib.error
import urllib.parse
import urllib.request
from typing import Any, Mapping, MutableMapping, Optional


class QrdotApiError(Exception):
    def __init__(self, code: str, message: str, status: int) -> None:
        super().__init__(message)
        self.code = code
        self.message = message
        self.status = status


def verify_webhook_signature(
    secret: str,
    raw_body: str | bytes,
    header: str,
    *,
    tolerance_sec: int = 300,
) -> bool:
    """Verify ``X-Qrdot-Signature: t=…,v1=…`` over ``f\"{t}.{rawBody}\"``."""
    body = raw_body if isinstance(raw_body, str) else raw_body.decode("utf-8")
    parts: dict[str, str] = {}
    for piece in header.split(","):
        if "=" not in piece:
            continue
        k, v = piece.strip().split("=", 1)
        parts[k] = v
    try:
        t = int(parts.get("t", ""))
    except ValueError:
        return False
    v1 = parts.get("v1")
    if not t or not v1:
        return False
    if abs(int(time.time()) - t) > tolerance_sec:
        return False
    expected = hmac.new(
        secret.encode("utf-8"),
        f"{t}.{body}".encode("utf-8"),
        hashlib.sha256,
    ).hexdigest()
    return hmac.compare_digest(expected, v1)


class Qrdot:
    def __init__(
        self,
        api_key: str,
        *,
        base_url: str = "https://api.qrdot.dev",
    ) -> None:
        if not (api_key.startswith("sk_test_") or api_key.startswith("sk_live_")):
            raise ValueError("api_key must start with sk_test_ or sk_live_")
        self.api_key = api_key
        self.base_url = base_url.rstrip("/")
        self.qr = _QrResource(self)
        self.assets = _AssetsResource(self)
        self.analytics = _AnalyticsResource(self)
        self.webhooks = _WebhooksResource(self)
        self.domains = _DomainsResource(self)

    def create_qr(
        self,
        payload: Mapping[str, Any],
        *,
        idempotency_key: str | None = None,
    ) -> Any:
        """Convenience — same as ``qr.create``."""
        return self.qr.create(payload, idempotency_key=idempotency_key)

    def request(
        self,
        method: str,
        path: str,
        body: Any | None = None,
        headers: Optional[Mapping[str, str]] = None,
    ) -> Any:
        data, status, _ct = self._raw(method, path, body, headers)
        if status == 204:
            return None
        if not data:
            return None
        return json.loads(data.decode("utf-8"))

    def request_bytes(
        self,
        method: str,
        path: str,
        body: Any | None = None,
    ) -> tuple[bytes, str]:
        data, _status, ct = self._raw(method, path, body, None)
        mime = (ct or "application/octet-stream").split(";")[0].strip()
        return data, mime

    def put_external(
        self,
        url: str,
        body: bytes,
        headers: Mapping[str, str],
    ) -> int:
        req = urllib.request.Request(url, data=body, method="PUT", headers=dict(headers))
        try:
            with urllib.request.urlopen(req, timeout=60) as res:
                return res.status
        except urllib.error.HTTPError as e:
            return e.code

    def _raw(
        self,
        method: str,
        path: str,
        body: Any | None,
        headers: Optional[Mapping[str, str]],
    ) -> tuple[bytes, int, str | None]:
        hdrs: MutableMapping[str, str] = {
            "Authorization": f"Bearer {self.api_key}",
            "Accept": "application/json",
        }
        data: bytes | None = None
        if body is not None:
            data = json.dumps(body).encode("utf-8")
            hdrs["Content-Type"] = "application/json"
        if headers:
            hdrs.update(headers)
        req = urllib.request.Request(
            f"{self.base_url}{path}",
            data=data,
            method=method,
            headers=dict(hdrs),
        )
        try:
            with urllib.request.urlopen(req, timeout=60) as res:
                return res.read(), res.status, res.headers.get("Content-Type")
        except urllib.error.HTTPError as e:
            raw = e.read()
            code = "internal"
            message = e.reason or "Request failed"
            try:
                parsed = json.loads(raw.decode("utf-8"))
                err = parsed.get("error") or {}
                code = err.get("code") or code
                message = err.get("message") or message
            except Exception:
                pass
            raise QrdotApiError(code, message, e.code) from None


class _QrResource:
    def __init__(self, client: Qrdot) -> None:
        self._c = client

    def create(
        self,
        payload: Mapping[str, Any],
        *,
        idempotency_key: str | None = None,
    ) -> Any:
        headers = (
            {"Idempotency-Key": idempotency_key} if idempotency_key else None
        )
        return self._c.request("POST", "/v1/qr", dict(payload), headers)

    def list(
        self,
        *,
        metadata: Mapping[str, str] | None = None,
        limit: int | None = None,
        cursor: str | None = None,
    ) -> Any:
        q: dict[str, str] = {}
        if metadata:
            for k, v in metadata.items():
                q[f"metadata[{k}]"] = v
        if limit is not None:
            q["limit"] = str(limit)
        if cursor:
            q["cursor"] = cursor
        qs = urllib.parse.urlencode(q)
        return self._c.request("GET", f"/v1/qr{('?' + qs) if qs else ''}")

    def batch(
        self,
        items: list[Mapping[str, Any]],
        *,
        idempotency_key: str | None = None,
    ) -> Any:
        headers = (
            {"Idempotency-Key": idempotency_key} if idempotency_key else None
        )
        return self._c.request(
            "POST", "/v1/qr/batch", {"items": list(items)}, headers
        )

    def get(self, qr_id: str) -> Any:
        return self._c.request("GET", f"/v1/qr/{urllib.parse.quote(qr_id)}")

    def update(self, qr_id: str, payload: Mapping[str, Any]) -> Any:
        return self._c.request(
            "PATCH", f"/v1/qr/{urllib.parse.quote(qr_id)}", dict(payload)
        )

    def delete(self, qr_id: str) -> None:
        self._c.request("DELETE", f"/v1/qr/{urllib.parse.quote(qr_id)}")

    def duplicate(self, qr_id: str) -> Any:
        return self._c.request(
            "POST", f"/v1/qr/{urllib.parse.quote(qr_id)}/duplicate"
        )

    def image(self, qr_id: str, format: str = "png") -> tuple[bytes, str]:
        return self._c.request_bytes(
            "GET",
            f"/v1/qr/{urllib.parse.quote(qr_id)}/image.{format}",
        )

    def export_images(
        self, ids: list[str], format: str = "png"
    ) -> tuple[bytes, str]:
        return self._c.request_bytes(
            "POST", "/v1/qr/export/images", {"ids": ids, "format": format}
        )


class _AssetsResource:
    def __init__(self, client: Qrdot) -> None:
        self._c = client

    def presign_logo(
        self,
        content_type: str,
        *,
        filename: str | None = None,
    ) -> Any:
        body: dict[str, Any] = {"content_type": content_type}
        if filename:
            body["filename"] = filename
        return self._c.request("POST", "/v1/assets/logo/presign", body)

    def complete_logo(
        self, asset_id: str, *, filename: str | None = None
    ) -> Any:
        body: dict[str, Any] = {}
        if filename:
            body["filename"] = filename
        return self._c.request(
            "POST",
            f"/v1/assets/logo/{urllib.parse.quote(asset_id)}/complete",
            body,
        )

    def list_logos(self) -> Any:
        return self._c.request("GET", "/v1/assets/logo")

    def get_logo(self, asset_id: str) -> Any:
        return self._c.request(
            "GET", f"/v1/assets/logo/{urllib.parse.quote(asset_id)}"
        )

    def delete_logo(self, asset_id: str) -> None:
        self._c.request(
            "DELETE", f"/v1/assets/logo/{urllib.parse.quote(asset_id)}"
        )

    def upload_logo(
        self,
        data: bytes,
        content_type: str,
        *,
        filename: str | None = None,
    ) -> Any:
        presign = self.presign_logo(content_type, filename=filename)
        status = self._c.put_external(
            presign["upload_url"], data, presign.get("headers") or {}
        )
        if status < 200 or status >= 300:
            raise QrdotApiError(
                "internal", f"Logo storage upload failed ({status})", status
            )
        return self.complete_logo(presign["asset_id"], filename=filename)


class _AnalyticsResource:
    def __init__(self, client: Qrdot) -> None:
        self._c = client

    def summary(
        self, *, from_: str | None = None, to: str | None = None
    ) -> Any:
        q: dict[str, str] = {}
        if from_:
            q["from"] = from_
        if to:
            q["to"] = to
        qs = urllib.parse.urlencode(q)
        return self._c.request(
            "GET", f"/v1/analytics/summary{('?' + qs) if qs else ''}"
        )

    def qr(
        self,
        qr_id: str,
        *,
        from_: str | None = None,
        to: str | None = None,
        group_by: str | None = None,
    ) -> Any:
        q: dict[str, str] = {}
        if from_:
            q["from"] = from_
        if to:
            q["to"] = to
        if group_by:
            q["group_by"] = group_by
        qs = urllib.parse.urlencode(q)
        return self._c.request(
            "GET",
            f"/v1/analytics/qr/{urllib.parse.quote(qr_id)}{('?' + qs) if qs else ''}",
        )

    def scans(
        self,
        qr_id: str,
        *,
        limit: int | None = None,
        starting_after: str | None = None,
    ) -> Any:
        q: dict[str, str] = {}
        if limit is not None:
            q["limit"] = str(limit)
        if starting_after:
            q["starting_after"] = starting_after
        qs = urllib.parse.urlencode(q)
        return self._c.request(
            "GET",
            f"/v1/analytics/qr/{urllib.parse.quote(qr_id)}/scans{('?' + qs) if qs else ''}",
        )


class _WebhooksResource:
    def __init__(self, client: Qrdot) -> None:
        self._c = client

    def create(self, payload: Mapping[str, Any]) -> Any:
        body = dict(payload)
        body.setdefault("events", ["qr.scanned"])
        return self._c.request("POST", "/v1/webhooks", body)

    def list(self) -> Any:
        return self._c.request("GET", "/v1/webhooks")

    def update(self, webhook_id: str, patch: Mapping[str, Any]) -> Any:
        return self._c.request(
            "PATCH",
            f"/v1/webhooks/{urllib.parse.quote(webhook_id)}",
            dict(patch),
        )

    def delete(self, webhook_id: str) -> None:
        self._c.request(
            "DELETE", f"/v1/webhooks/{urllib.parse.quote(webhook_id)}"
        )

    def test(
        self, webhook_id: str, payload: Mapping[str, Any] | None = None
    ) -> Any:
        return self._c.request(
            "POST",
            f"/v1/webhooks/{urllib.parse.quote(webhook_id)}/test",
            dict(payload or {}),
        )

    def list_deliveries(
        self, webhook_id: str, *, limit: int | None = None
    ) -> Any:
        q: dict[str, str] = {}
        if limit is not None:
            q["limit"] = str(limit)
        qs = urllib.parse.urlencode(q)
        return self._c.request(
            "GET",
            f"/v1/webhooks/{urllib.parse.quote(webhook_id)}/deliveries{('?' + qs) if qs else ''}",
        )

    def replay(
        self, webhook_id: str, payload: Mapping[str, Any]
    ) -> Any:
        return self._c.request(
            "POST",
            f"/v1/webhooks/{urllib.parse.quote(webhook_id)}/replay",
            dict(payload),
        )

    @staticmethod
    def verify_signature(
        secret: str,
        raw_body: str | bytes,
        header: str,
        *,
        tolerance_sec: int = 300,
    ) -> bool:
        return verify_webhook_signature(
            secret, raw_body, header, tolerance_sec=tolerance_sec
        )


class _DomainsResource:
    def __init__(self, client: Qrdot) -> None:
        self._c = client

    def create(self, payload: Mapping[str, Any]) -> Any:
        return self._c.request("POST", "/v1/domains", dict(payload))

    def list(self) -> Any:
        return self._c.request("GET", "/v1/domains")

    def get(self, domain_id: str) -> Any:
        return self._c.request(
            "GET", f"/v1/domains/{urllib.parse.quote(domain_id)}"
        )

    def dns(self, domain_id: str) -> Any:
        return self._c.request(
            "GET", f"/v1/domains/{urllib.parse.quote(domain_id)}/dns"
        )

    def dns_provider(self, domain_id: str) -> Any:
        return self._c.request(
            "GET",
            f"/v1/domains/{urllib.parse.quote(domain_id)}/dns-provider",
        )

    def domain_connect_start(self, domain_id: str) -> Any:
        return self._c.request(
            "POST",
            f"/v1/domains/{urllib.parse.quote(domain_id)}/domain-connect/start",
        )

    def forward_dns(
        self, domain_id: str, payload: Mapping[str, Any]
    ) -> Any:
        return self._c.request(
            "POST",
            f"/v1/domains/{urllib.parse.quote(domain_id)}/dns/forward",
            dict(payload),
        )

    def refresh(self, domain_id: str) -> Any:
        return self._c.request(
            "POST", f"/v1/domains/{urllib.parse.quote(domain_id)}/refresh"
        )

    def set_default(self, domain_id: str) -> Any:
        return self._c.request(
            "POST", f"/v1/domains/{urllib.parse.quote(domain_id)}/default"
        )

    def delete(self, domain_id: str) -> None:
        self._c.request(
            "DELETE", f"/v1/domains/{urllib.parse.quote(domain_id)}"
        )
