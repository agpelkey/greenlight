package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

type envelope map[string]interface{}

func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst interface{}) error {

    // use http.MaxBytesReader to limit the size of the request body to 1MB
    maxBytes := 1_048_576
    r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

    // initialize the json.Decoder, and call the DisallowUnknownFields() method on it
    // before decoding. This meands that if the JSON from the client now includes
    // any field which cannot be mapped to the target destination, the decoder 
    // will return an error instead of just ignoring the field.
    dec := json.NewDecoder(r.Body)
    dec.DisallowUnknownFields()

    // decode the request body into the target destination
    err := dec.Decode(dst)
    if err != nil {
        // if there is an error decoding, start the triage...
        var syntaxError *json.SyntaxError
        var unmarshalTypeError *json.UnmarshalTypeError
        var invalidUnmarshalError *json.InvalidUnmarshalError

        switch {
        // use the errors.As() function to check whether the error has the type 
        // *json.SyntaxError. If it does, then return a plain-english error message 
        // which includes the location of the problem
        case errors.As(err, &syntaxError):
            return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)

        // In some circumstances Decode() may also return an io.ErrUnexpectedEOF error
        // for syntax erros in the JSON. So we check for this using errors.Is()
        // and return a generic error message.
        case errors.Is(err, io.ErrUnexpectedEOF):
            return errors.New("body contains badly-formed JSON")

        // likewise, catch any *json.UnmarshalTypeError errors. These occurr when the json value 
        // is the wrong type for the target destination. If the error relates to a specific field, 
        // then we include that in our error message to make it easier for the client to debug.
        case errors.As(err, &unmarshalTypeError):
            if unmarshalTypeError.Field != "" {
                return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
            }
            return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)

        // An io.EOF error will be returned by Decode() if the request body is empty
        // We check for this with errors.Is() and return a plain-english error message instead.
        case errors.Is(err, io.EOF):
            return errors.New("body must not be empty")
            
        // a json.invalidUnmarshalError error will be returned if we pass a non-nil pointer to Decode().
        // We catch this and panic, rather than returning an error to our handler.
        case errors.As(err, &invalidUnmarshalError):
            panic(err)

        // for anything else, return the error message as is.
        default:
            return err
        }

    }

    // Call Decode() again, using a pointer to an empty anonymous struct as the 
    // destination. If the request body only contained a single JSON value
    // this will return an io.EOF error. So if we get anything else, we know
    // that there is additional data in the request body and we return our own
    // custom error message.
    err = dec.Decode(&struct{}{})
    if err != io.EOF {
       return errors.New("body must only contain a single JSON value") 
    }
    
    return nil
}

func (app *application) writeJSON(w http.ResponseWriter, status int, data envelope, header http.Header) error {
    // Encode the data to JSON, returning the error if there was one
    js, err := json.MarshalIndent(data, "", "\t")
    if err != nil {
        return err
    }

    // append a new line to make it easier to view in terminal applications
    js = append(js, '\n')

    for key, value := range header {
        w.Header()[key] = value
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    w.Write(js)

    return nil 
}

func (app *application) readIDParam(r *http.Request) (int64, error) {
    params := httprouter.ParamsFromContext(r.Context())

    id, err := strconv.ParseInt(params.ByName("id"), 10, 64)
    if err != nil || id < 1{
        return 0, errors.New("invalid id parameter")

    }

    return id, nil
}
