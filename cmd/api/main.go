package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

const version = "1.0.0"

type config struct {
    port int
    env string
}

type application struct {
    config config
    logger *log.Logger 
}

func main() {
    // instantiate config
    var cfg config

    // Read in the value for port and environment
    flag.IntVar(&cfg.port, "port", 8080, "API Server Port")
    flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
    flag.Parse()

    // initialize logger which writes messages to STDOUT
    // prefix logger with current date and time
    logger := log.New(os.Stdout, "", log.Ldate | log.Ltime)

    // Declare an instance of the application struct, containing the config struct and the logger
    app := application{
        config: cfg,
        logger: logger,
    }

    // Declare an HTTP server with some timeout settings
    srv := &http.Server{
        Addr: fmt.Sprintf(":%d", cfg.port),
        Handler: app.routes(),
        IdleTimeout: time.Minute,
        ReadTimeout: 10 * time.Second,
        WriteTimeout: 30 * time.Second,
    }


    // Start server
    logger.Println("starting %s server on port %s", cfg.env, cfg.port)
    err := srv.ListenAndServe()
    logger.Fatal(err)
}




















