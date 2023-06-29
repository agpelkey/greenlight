package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/agpelkey/greenlight/internal/data"
	"github.com/agpelkey/greenlight/internal/jsonlog"
	_ "github.com/lib/pq"
)

const version = "1.0.0"

type config struct {
    port int
    env string
    db struct {
        dsn string
        maxOpenConns int 
        maxIdleConns int
        maxIdleTime string 
    }
}

type application struct {
    config config
    logger *jsonlog.Logger
    models data.Models
}

func main() {
    // instantiate config
    var cfg config

    // Read in the value for port and environment
    flag.IntVar(&cfg.port, "port", 8080, "API Server Port")
    flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")

    flag.StringVar(&cfg.db.dsn, "db-dsn", "user=greenlight password=greenlight dbname=greenlight sslmode=disable", "PostgreSQL DSN")

    // Read the connection pool settings from the command-line flags into
    // the config struct. Note the default values being passed here
    flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
    flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
    flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connections idle time")

    flag.Parse()

    // initialize logger which writes messages to STDOUT
    // prefix logger with current date and time
    logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)
    
    db, err := openDB(cfg)
    if err != nil {
        logger.PrintFatal(err, nil)
    }
    
    defer db.Close()

    logger.PrintInfo("database connection pool established", nil)

    // Declare an instance of the application struct, containing the config struct and the logger
    app := &application{
        config: cfg,
        logger: logger,
        models: data.NewModels(db),
    }

    // Declare an HTTP server with some timeout settings
    srv := &http.Server{
        Addr: fmt.Sprintf(":%d", cfg.port),
        Handler: app.routes(),
        ErrorLog: log.New(logger, "", 0),
        IdleTimeout: time.Minute,
        ReadTimeout: 10 * time.Second,
        WriteTimeout: 30 * time.Second,
    }


    // Start server
    logger.PrintInfo("starting server", map[string]string{
        "addr": srv.Addr,
        "env": cfg.env,
    })

    err = srv.ListenAndServe()
    logger.PrintFatal(err, nil)
}


func openDB(cfg config) (*sql.DB, error) {
    
    // use sql.open to create connection pool
    db, err := sql.Open("postgres", cfg.db.dsn)
    if err != nil {
        return nil, err
    }

    // Set the maximum number of open (in-use + idle) connections in the pool. 
    // Passing a value that is less than or equal to zero will mean there is no limit
    db.SetMaxOpenConns(cfg.db.maxOpenConns)

    // Set the maximum number of idle connections in the pool.
    // Zero means there is no limit
    db.SetMaxIdleConns(cfg.db.maxIdleConns)

    // Use time.ParseDuration() function to convert the idle timeout duration string
    // to a time.Duration type
    duration, err := time.ParseDuration(cfg.db.maxIdleTime)
    if err != nil {
        return nil, err
    }
    
    // Set the maximum idle timeout
    db.SetConnMaxIdleTime(duration)

    // create context with a 5 second timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // use pingcontext to establish connection pool, passing in the 
    // context as an argument. If the connection cannot be made,
    // the connection will timeout in 5 seconds.
    err = db.PingContext(ctx)
    if err != nil {
        return nil, err
    }

    return db, nil
}




















