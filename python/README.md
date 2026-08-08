# qrdot (Python)

Official Python client for the [QR.](https://qrdot.dev) API.

```bash
pip install qrdot
```

## Quick start

```python
from qrdot import Qrdot

client = Qrdot(api_key="sk_test_…")  # or sk_live_…

qr = client.qr.create({
    "targetUrl": "https://example.com",
    "name": "Launch",
    "style": {"dark_color": "#0F766E", "light_color": "#FFFFFF"},
})

png, _mime = client.qr.image(qr["id"], "png")
zip_bytes, _ = client.qr.export_images([qr["id"]], "png")

batch = client.qr.batch([
    {"targetUrl": "https://event.example/seat", "name": "Table 1"},
])

# Starter+: logo upload (presign → S3 PUT → complete)
# logo = client.assets.upload_logo(open("mark.png", "rb").read(), "image/png", filename="mark.png")

from qrdot import verify_webhook_signature
ok = verify_webhook_signature(secret, raw_body, request.headers["X-Qrdot-Signature"])
```

## Docs

- Guides: https://qrdot.dev/docs/
- OpenAPI: https://api.qrdot.dev/v1/openapi.json
- Node SDK: `npm i @qrdot/sdk`
