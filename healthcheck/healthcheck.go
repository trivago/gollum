// This package provides a simple health check server.
//
// The package lets you register arbitrary health check endpoints by
// providing a name (== URL path) and a callback function.
//
// The health check server listens for HTTP requests on a given port;
// probes are executed by issuing requests to the endpoints' named
// URL paths.
//
// The package works as a "singleton" with just one server in order to
// avoid cluttering the main program by passing handles around.
package healthcheck

import (
	"net/http"
	"fmt"
	"bytes"
)

// Code wishing to get probed by the health-checker needs to provide this callback
type CallbackFunc func () (code int, body string)

// The HTTP server
var server *http.Server

// The HTTP request multiplexer
var serveMux *http.ServeMux

// List of endpoints known by the server
var endpoints map[string]CallbackFunc

// Configures the health check server
//
// listenAddr: an address understood by http.ListenAndServe(), e.g. ":8008"
func Configure(listenAddr string) {
	// Create our request multiplexer
	serveMux = http.NewServeMux()

	// Create the HTTP server
	server = &http.Server{
		Addr:    listenAddr,
		Handler: serveMux,
	}

	// Add default wildcard handler
	serveMux.HandleFunc("/",
		func(responseWriter http.ResponseWriter, httpRequest *http.Request) {
			path := httpRequest.URL.Path

			// Handle "/": list all our registered endpoints
			if path == "/" {
				fmt.Fprintf(responseWriter, "/_ALL_\n")

				// TBD: Collate
				for endpointPath, _ := range endpoints {
					fmt.Fprintf(responseWriter, "%s\n", endpointPath)
				}
				return
			}

			// Default action: 404
			responseWriter.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(responseWriter, "Path not found\n")
			return
		},
	)

	// Add magical "/_ALL" handler
	serveMux.HandleFunc("/_ALL_",
		func(responseWriter http.ResponseWriter, httpRequest *http.Request) {
			// Response code needs to be set before writing the response body,
			// so we need to pool the body temporarily into resultBody
			var resultCode = 200
			var resultBody bytes.Buffer

			// Call all endpoints sequentially
			// (TBD: if there were _a lot_ of these we could do them in parallel)
			for endpointPath, callback := range endpoints {
				// Call the callback
				code, body := callback()
				// Append path, code, body to response body
				fmt.Fprintf(&resultBody,
					"%s %d %s",
					endpointPath,
					code,
					body,
				)
				// Naive assumption: the bigger the code the more serious it is
				if code > resultCode {
					resultCode = code
				}
			}

			// Set HTTP response code
			responseWriter.WriteHeader(resultCode)

			// Write HTTP response body
			// (TBD: more efficient way, buffer -> writer?)
			fmt.Fprintf(responseWriter, resultBody.String())
		},
	)

	// Initialize the enpoint list
	endpoints = make(map[string]CallbackFunc)

	// Add a static "ping" endpoint
	AddEndpoint("/ping", func()(code int, body string){
		return 200, "PONG\n"
	})

	// Debugging
	//AddEndpoint("/ping", func()(code int, body string){
	//	return 400, "FAILPONG\n"
	//})

	// Debugging
	//AddEndpoint("/fail", func()(code int, body string){
	//	return 500, "FAIL\n"
	//})
}

// Registers an endpoint with the health checker
func AddEndpoint(urlPath string, callback CallbackFunc){
	// Check parameters - reserved paths
	for _, path := range []string{"", "/", "/_ALL_"} {
		if urlPath == path {
			panic(fmt.Sprintf(
				"ERROR: Health check path \"%s\" is reserved", path))
		}
	}

	// Check parameters - registered paths
	_, exists := endpoints[urlPath]
	if exists {
		panic(fmt.Sprintf(
			"ERROR: Health check endpoint \"%s\" already registered", urlPath))
	}

	// Register the HTTP route
	serveMux.HandleFunc(urlPath,
		func(responseWriter http.ResponseWriter, httpRequest *http.Request){
			// Call the callback
			code, body := callback()
			// Set HTTP response code
			responseWriter.WriteHeader(code)
			// Write HTTP response body
			fmt.Fprintf(responseWriter, body)
		},
	)

	// Store the endpoint
	endpoints[urlPath] = callback
}

// Starts the HTTP server
func Start(){
	err := server.ListenAndServe()

	if err != nil {
		panic(err.Error())
	}
}

// Cleanup?
func Stop(){


}
