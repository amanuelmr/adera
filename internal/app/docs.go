package app

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"

	apidocs "github.com/adera-platform/backend/api"
)

// Swagger UI comes from unpkg at an exact version, and the browser checks
// each file against its hash (subresource integrity), so a changed file on
// the CDN is refused rather than run on the API's origin. To upgrade, change
// the version and recompute both hashes:
//
//	curl -sL https://unpkg.com/swagger-ui-dist@VERSION/FILE | openssl dgst -sha384 -binary | openssl base64 -A
const (
	swaggerBase   = "https://unpkg.com/swagger-ui-dist@5.33.0/"
	swaggerCSSSRI = "sha384-Ov4/wv3j2bmct8cDc5X4ngJZohVPzEmc6uDPH8WeljUxO5vtoykvMEfbu9Vh6RaW"
	swaggerJSSRI  = "sha384-YDALVcy8kj8yltLBVi1vBiBAUqdxvus673gM8XKwiy6aDUJFXivF/KCufekjYbVf"
)

// swaggerInit starts the UI. The CSP allows exactly this script by its hash,
// so no other inline script can run.
const swaggerInit = `
    window.ui = SwaggerUIBundle({
      url: "/openapi.yaml",
      dom_id: "#swagger-ui",
      deepLinking: true,
      persistAuthorization: true,
      displayRequestDuration: true
    });
  `

var swaggerHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Adera API Docs</title>
  <link rel="stylesheet" href="` + swaggerBase + `swagger-ui.css" integrity="` + swaggerCSSSRI + `" crossorigin="anonymous">
  <style>
    body { margin: 0; background: #fff; }
    .swagger-ui .topbar { display: none; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="` + swaggerBase + `swagger-ui-bundle.js" integrity="` + swaggerJSSRI + `" crossorigin="anonymous"></script>
  <script>` + swaggerInit + `</script>
</body>
</html>
`

var swaggerCSP = func() string {
	sum := sha256.Sum256([]byte(swaggerInit))
	return "default-src 'none'; script-src https://unpkg.com 'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'; " +
		"style-src 'self' 'unsafe-inline' https://unpkg.com; img-src data:; connect-src 'self'; frame-ancestors 'none'"
}()

func registerDocs(mux *http.ServeMux) {
	mux.HandleFunc("GET /openapi.yaml", serveOpenAPI)
	mux.HandleFunc("GET /docs", serveSwagger)
	mux.HandleFunc("GET /docs/", serveSwagger)
}

func serveOpenAPI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(apidocs.OpenAPIYAML)
}

func serveSwagger(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", swaggerCSP)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(swaggerHTML))
}
