package data

import (
	"database/sql"
	"errors"
)

// define a custom ErrRecordNotFound error. Return this
// from our Get() method when looking up a movie that
// doesnt exist in our database.
var (
    ErrRecordNotFound = errors.New("record not found")
    ErrEditConflict = errors.New("edit conflict")
)


// Create a models struct which wraps the MovieModel.
// Add other models to this, like a UserModel and PermissionModel
type Models struct {
    Movies MovieModel
}

// for ease of use, we also add a New() method which returns a Models
// struct containing the initialized MovieModel.
func NewModels(db *sql.DB) Models {
    return Models{
        Movies: MovieModel{DB: db},
    }
}
