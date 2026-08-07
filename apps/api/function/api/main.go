// Command api is the Netlify Function entry point for the TrustCheck API. It
// reuses the router from internal/server and adapts it to the AWS Lambda
// invocation model used by Netlify Functions, so the deployed API behaves
// identically to the local development server (cmd/api).
//
// The function is named "api" and is mounted behind the /api/* rewrite in
// netlify.toml, which routes every /api/** request to it on the same origin
// as the frontend.
package main

import (
	"context"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/pamierin/trustcheck/apps/api/internal/server"
)

// apiPrefix is the URL prefix under which the API is mounted on Netlify (see
// the /api/* redirect in netlify.toml). It is registered on every route so
// the Swagger UI can resolve the OpenAPI specification at a stable absolute
// path.
const apiPrefix = "/api"

var ginLambda *ginadapter.GinLambda

func init() {
	ginLambda = ginadapter.New(server.NewRouter(apiPrefix))
}

// handler translates the Netlify/Lambda proxy event into an HTTP request and
// runs it through the router.
func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	req.Path = normalizePath(req.Path)
	return ginLambda.ProxyWithContext(ctx, req)
}

// normalizePath maps whatever path Netlify delivers to the route the router
// expects. When the function is reached through the /api/* rewrite, the event
// path is the original request path (/api/verify); when called directly it is
// the raw function path (/.netlify/functions/api/verify). Both are normalized
// to /api/verify.
func normalizePath(path string) string {
	const rawPrefix = "/.netlify/functions/api"

	if strings.HasPrefix(path, rawPrefix) {
		path = strings.TrimPrefix(path, rawPrefix)
	}
	if !strings.HasPrefix(path, apiPrefix) {
		path = apiPrefix + path
	}
	return path
}

func main() {
	lambda.Start(handler)
}
