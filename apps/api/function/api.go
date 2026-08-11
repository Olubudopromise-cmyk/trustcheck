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
	"log"
	"net/http"
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
	log.Println("Initializing API router...")
	ginLambda = ginadapter.New(server.NewRouter(apiPrefix))
	log.Println("API router initialized successfully")
}

// handler translates the Netlify/Lambda proxy event into an HTTP request and
// runs it through the router. Any panic is recovered here so a single bad
// request can never take the whole function down or surface as a platform
// 502: the panic is written to the function logs (where it can actually be
// diagnosed) and a clean 500 JSON response is returned instead.
func handler(ctx context.Context, req events.APIGatewayProxyRequest) (resp events.APIGatewayProxyResponse, err error) {
	log.Printf("Handling request: %s %s", req.HTTPMethod, req.Path)
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[panic] recovered in handler for %s %s: %v", req.HTTPMethod, req.Path, r)
			resp = events.APIGatewayProxyResponse{
				StatusCode: http.StatusInternalServerError,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body: `{"error":"internal error"}`,
			}
		}
	}()

	req.Path = normalizePath(req.Path)
	log.Printf("Normalized path: %s", req.Path)
	resp, err = ginLambda.ProxyWithContext(ctx, req)
	if err != nil {
		// If the context was cancelled (Lambda deadline or client disconnect),
		// the adapter may return a partial/empty response but the error is
			// just context cancellation — not a real failure. Return whatever
			// response we have instead of propagating the error, which would
			// cause the platform to surface a 502 for a normal timeout.
		if ctx.Err() != nil {
			log.Printf("[warn] context expired for %s %s: %v (returning partial response)", req.HTTPMethod, req.Path, ctx.Err())
			if resp.Body == "" {
				resp = events.APIGatewayProxyResponse{
					StatusCode: http.StatusGatewayTimeout,
					Headers:    map[string]string{"Content-Type": "application/json"},
					Body:       `{"error":"request timed out"}`,
				}
			}
			return resp, nil
		}
		log.Printf("[error] handler failed for %s %s: %v", req.HTTPMethod, req.Path, err)
	}
	return resp, err
}

// normalizePath maps whatever path Netlify delivers to the route the router
// expects. When the function is reached through the /api/* redirect, the
// event path is the original request path (/api/verify); when called directly
// it is the raw function path (/.netlify/functions/api/verify).
// Both are normalized to /api/verify.
func normalizePath(path string) string {
	const rawPrefix = "/.netlify/functions/api"
	const publicPrefix = "/api"

	if strings.HasPrefix(path, rawPrefix) {
		path = strings.TrimPrefix(path, rawPrefix)
	}
	if strings.HasPrefix(path, publicPrefix) {
		path = strings.TrimPrefix(path, publicPrefix)
	}
	if !strings.HasPrefix(path, apiPrefix) {
		path = apiPrefix + path
	}
	return path
}

func main() {
	lambda.Start(handler)
}
