package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() *httprouter.Router {

    router := httprouter.New()

    // http.handlerFunc acts as an adapter to convert notFoundResponse() to an http.Handler
    // This is then set as the custome error handler for 404 Not Found responses from the router
    router.NotFound = http.HandlerFunc(app.notFoundResponse)

    // Likewise, methodNotAllowedResponse is set as the custom error handler for 405 Method Not Allowed
    router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

    router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.handleHealthCheck)


    router.HandlerFunc(http.MethodGet, "/v1/movies", app.handleListMovies)
    router.HandlerFunc(http.MethodPost, "/v1/movies", app.handleCreateMovie)
    router.HandlerFunc(http.MethodGet, "/v1/movies/:id", app.handleGetMovieByID)
    router.HandlerFunc(http.MethodPatch, "/v1/movies/:id", app.handleUpdateMovie)
    router.HandlerFunc(http.MethodDelete, "/v1/movies/:id", app.handleDeleteMovie)

    return router

}
