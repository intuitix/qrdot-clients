# qrdot (Ruby)

Official Ruby client for the [QR.](https://qrdot.dev) API.

```bash
gem install qrdot
# or: bundle add qrdot
```

## Quick start

```ruby
require "qrdot"

client = Qrdot::Client.new(ENV.fetch("QRDOT_API_KEY"))

qr = client.qr.create({
  "targetUrl" => "https://example.com",
  "name" => "Launch",
  "style" => { "dark_color" => "#0F766E", "light_color" => "#FFFFFF" },
})

png, mime = client.qr.image(qr["id"], "png")
zip, = client.qr.export_images([qr["id"]], "png")

ok = Qrdot.verify_webhook_signature(secret, raw_body, request.get_header("HTTP_X_QRDOT_SIGNATURE"))
```

## Docs

- Guides: https://qrdot.dev/docs/
- OpenAPI: https://api.qrdot.dev/v1/openapi.json
