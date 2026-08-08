# qrdot (Go)

Official Go client for the [QR.](https://qrdot.dev) API.

Published from the public mirror [`intuitix/qrdot-clients`](https://github.com/intuitix/qrdot-clients) (the product monorepo stays private).

```bash
go get github.com/intuitix/qrdot-clients/go@v0.1.1
```

## Quick start

```go
package main

import (
	"os"

	qrdot "github.com/intuitix/qrdot-clients/go"
)

func main() {
	client, err := qrdot.New(os.Getenv("QRDOT_API_KEY"))
	if err != nil {
		panic(err)
	}

	qr, err := client.QR.Create(map[string]any{
		"targetUrl": "https://example.com",
		"name":      "Launch",
		"style": map[string]any{
			"dark_color":  "#0F766E",
			"light_color": "#FFFFFF",
		},
	}, "")
	if err != nil {
		panic(err)
	}

	png, _, err := client.QR.Image(qr["id"].(string), "png")
	_ = png

	_, _, _ = client.QR.ExportImages([]string{qr["id"].(string)}, "png")

	ok := qrdot.VerifyWebhookSignature(secret, rawBody, header, 300)
	_ = ok
}
```

## Docs

- Guides: https://qrdot.dev/docs/
- OpenAPI: https://api.qrdot.dev/v1/openapi.json
- Source mirror: https://github.com/intuitix/qrdot-clients
