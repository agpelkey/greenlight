package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/agpelkey/greenlight/internal/validator"
	"github.com/lib/pq"
)

type MovieModel struct {
    DB *sql.DB
}

func (m MovieModel) GetAll(title string, genres []string, filters Filters) ([]*Movie, Metadata, error) {
    // Construct the SQL query to retreive all movie records
    query := fmt.Sprintf(`
    SELECT count(*) OVER(), id, created_at, title, year, runtime, genres, version 
    FROM movies 
    WHERE (to_tsvector('simple', title) @@ plainto_tsquery('simple', $1) OR $1 = '') 
    AND (genres @> $2 OR $2 = '{}') 
    ORDER BY %s %s, id ASC
    LIMIT $3 OFFSET $4`, filters.sortColumn(), filters.sortDirection())
        
    // Create context with 3 second timeout
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()

    // Our SQL query now has quite a few placeholder parameters, lets collect the
    // values for the placeholders in a slice. Notice here how we call the limit()
    // and offset() methods on the Filters struct to get the appropriate values for the
    // LIMIT and OFFSET clauses.
    args := []interface{}{title, pq.Array(genres), filters.limit(), filters.offset()}

    // Use QueryContext() to execute the query. This returns a sql.Rows resultset
    // containing the result
    rows, err := m.DB.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, Metadata{}, err
    }

    defer rows.Close()

    // Initialize an empty slice to hold the movie data
    totalRecords := 0
    movies := []*Movie{}

    // Use rows.Next to iterate through the rows in the resultset
    for rows.Next() {
        var movie Movie

        err := rows.Scan(
            &totalRecords,
            &movie.ID,
            &movie.CreatedAt,
            &movie.Title,
            &movie.Year,
            &movie.Runtime,
            pq.Array(&movie.Genres),
            &movie.Version,
        )
        if err != nil {
            return nil, Metadata{}, err
        }

        movies = append(movies, &movie)
    }
    if err = rows.Err(); err != nil {
       return nil, Metadata{}, err 
    }

    metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)

    return movies, metadata, nil
}

func (m MovieModel) Insert(movie *Movie) error {
    // define the sql query for inserting a new record in the movies table 
    // and returning the system-generated data.
    query := `INSERT INTO movies (title, year, runtime, genres) VALUES
    ($1, $2, $3, $4) RETURNING id, created_at, version`

    // create an args slice containing the values for the placeholder parameters
    // from thje movie struct. Declaring this slice immediately next to our SQL query
    // helps to make it nice and clear *what values are being used where* in the query
    args := []interface{}{movie.Title, movie.Year, movie.Runtime, pq.Array(movie.Genres)}

    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()

    // use the QueryRow() method to execute the SQL query on our connection pool,
    // passing in the args slice as a variadic parameter and scanning the system-
    // generated id, created_at, and version values into the movie struct
    return m.DB.QueryRowContext(ctx, query, args...).Scan(&movie.ID, &movie.CreatedAt, &movie.Version)
}

func (m MovieModel) Get(id int64) (*Movie, error) {
    // The PostgreSQL bigseriral type that we're using for the movie id
    // starts auto-incrementin at 1 by default, so we know that no movies will have
    // ID values less than that. To avoid making an unnecessary databse call, we take
    // a shortcut and return an ErrRecordNotFound error straight away
    if id < 1 {
        return nil, ErrRecordNotFound
    }

    // Define the SQL query for retrieving the movie data.
    query := `SELECT id, created_at, title, year, runtime, genres, version 
    FROM movies
    WHERE id = $1`

    // Declare a movie struct to hold the data returned by the query
    var movie Movie

    // Use the context.WithTimeout() function to create a context.Context which
    // carries a 3-second timeout deadline. Note that we're using the empty context.Background()
    // as the 'parent' context
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

    // importantly, user defer to make sure we cancel the context before the Get() method returns
    defer cancel()

    // Execute the query using the QueryRow() method, passing in the provided id value
    // as a placeholder parameter, and scan the response data into the fields of the
    // Movie struct. Importantly, notice that we need to convert the scan target for the
    // genres column using the pq.Arrary() adpater function again.
    err := m.DB.QueryRowContext(ctx, query, id).Scan(
        &movie.ID,
        &movie.CreatedAt,
        &movie.Title,
        &movie.Year,
        &movie.Runtime,
        pq.Array(&movie.Genres),
        &movie.Version,
    )

    // Handler any errors. If there was no matching movie found, Scan() will return
    // a sql.ErrNoRows error. We check for this and return our custom ErrRecordNotFound
    // error instead.
    if err != nil {
        switch {
        case errors.Is(err, sql.ErrNoRows):
            return nil, ErrRecordNotFound
        default:
            return nil, err
        }
    }

    // Otherwise, return a pointer to the Movie struct
    return &movie, nil

}

func (m MovieModel) Update(movie *Movie) error {
    // Declare the SQL query for updating the record and returning the new version number
    query := `
        UPDATE movies
        SET title = $1, year = $2, runtime = $3, genres = $4, version = version + 1
        WHERE id = $5 AND version = $6
        RETURNING version`

    // Create an args slice containing the values for the placeholder parameters
    args := []interface{}{
        movie.Title,
        movie.Year,
        movie.Runtime,
        pq.Array(movie.Genres),
        movie.ID,
        movie.Version,
    }

    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()

    // Execute the SQL query. If no matching row could be found, we know the movie version has changed (or the record has been deleted)
    // and we return our custom ErrEditConflict error.
    err := m.DB.QueryRowContext(ctx, query, args...).Scan(&movie.Version)
    if err != nil {
        switch {
        case errors.Is(err, sql.ErrNoRows):
            return ErrEditConflict
        default:
            return err
        }
    }

    return nil 
}

func (m MovieModel) Delete(id int64) error {
    // Return an ErrRecordNotFound error if the movie ID is less than 1
    if id < 1 {
        return ErrRecordNotFound
    }

    // Construct the SQL query to delete the record
    query := `
        DELETE FROM movies
        WHERE id = $1`

    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()

    // Execute the SQL query using the Exec() method, passing in the id variable as
    // the value for the placeholder parameter. The Exec() method returns a sql.Result
    // object.
    result, err := m.DB.ExecContext(ctx, query, id)
    if err != nil {
        return err
    }

    // Call the RowsAffected() method on the sql.Result object to get the number of rows
    // affected by the query
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return err
    }

    // If no rows were affected, we know that the movies table didnt contain a record 
    // with the provided ID at the moment we tried to delete it. In that case we return
    // an ErrRecordNotFound error.
    if rowsAffected == 0 {
        return ErrRecordNotFound
    }

    return nil 
}

type Movie struct {
    ID int64 `json:"id"` 
    CreatedAt time.Time `json:"-"`
    Title string `json:"title"`
    Year int32 `json:"year,omitempty"`
    Runtime Runtime `json:"runtime,omitempty,string"`
    Genres []string `json:"genres,omitempty"`
    Version int32  `json:"version"`
}

func ValidateMovie(v *validator.Validator, movie *Movie) {
v.Check(movie.Title != "", "title", "must be provided")
v.Check(len(movie.Title) <= 500, "title", "must not be more than 500 bytes long")
v.Check(movie.Year != 0, "year", "must be provided")
v.Check(movie.Year >= 1888, "year", "must be greater than 1888")
v.Check(movie.Year <= int32(time.Now().Year()), "year", "must not be in the future")
v.Check(movie.Runtime != 0, "runtime", "must be provided")
v.Check(movie.Runtime > 0, "runtime", "must be a positive integer")
v.Check(movie.Genres != nil, "genres", "must be provided")
v.Check(len(movie.Genres) >= 1, "genres", "must contain at least 1 genre")
v.Check(len(movie.Genres) <= 5, "genres", "must not contain more than 5 genres")
v.Check(validator.Unique(movie.Genres), "genres", "must not contain duplicate values")
}
