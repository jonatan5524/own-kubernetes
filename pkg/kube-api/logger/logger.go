package kubeapi_logger

import (
	"log"
	"time"

	restful "github.com/emicklei/go-restful/v3"
)

func LoggerMiddleware(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	start := time.Now()

	// Log request information
	log.Printf("Incoming request: %s %s", req.Request.Method, req.Request.URL.Path)

	// Process the request
	chain.ProcessFilter(req, resp)

	// Log response information
	log.Printf("Outgoing response: %s %d %s", req.Request.Method, resp.StatusCode(), time.Since(start))
}
