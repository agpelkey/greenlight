package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/agpelkey/greenlight/internal/data"
	"github.com/agpelkey/greenlight/internal/validator"
)

func (app *application) handleCreateMovie(w http.ResponseWriter, r *http.Request) {

    var input struct {
        Title string `json:"title"`
        Year int32 `json:"year"`
        Runtime data.Runtime`json:"runtime"`
        Genres []string `json:"genres"`
    }

    // use readJSON() to decode the request body into the input struct.
    // If this returns an error we send the client the error message 
    // along with a 400 Bad Request status code, just like before.
    err := app.readJSON(w, r, &input)
    if err != nil {
        app.badRequestResponse(w, r, err)
        return
    }

    // copy the values from the input struct to a new movie struct
    movie := &data.Movie{
        Title: input.Title,
        Year: input.Year,
        Runtime: input.Runtime,
        Genres: input.Genres,
    }

    v := validator.New()

    // call the ValidateMovie() function and return a response containing the errors
    // if any checks fail
    if data.ValidateMovie(v, movie); !v.Valid() {
        app.failedValidationResponse(w, r, v.Errors)
        return
    }

    fmt.Fprintf(w, "%+v\n", input)
    
}
func (app *application) handleGetMovieByID(w http.ResponseWriter, r *http.Request) {

    id, err := app.readIDParam(r)
    if err != nil {
        http.NotFound(w, r)
        return
    }

    movie := data.Movie{
        ID: id,
        CreatedAt: time.Now(),
        Title: "Casablanca",
        Runtime: 102,
        Genres: []string{"drama", "romance", "war"},
        Version: 1,
    }

    err = app.writeJSON(w, http.StatusOK, envelope{"movie": movie}, nil)
    if err != nil {
        app.serverErrorResponse(w, r, err)
    }

}
