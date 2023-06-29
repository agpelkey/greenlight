package jsonlog

import (
	"encoding/json"
	"io"
	"os"
	"runtime/debug"
	"sync"
	"time"
)

// define a level type to represent the severity level for a log entry
type Level int8

// Initialize constant which represent a specific severity level.
// We use the iota keyword as a shortcut to assign successive integer values 
// to the constants

const (
    LevelInfo Level = iota   // Has the value of 0
    LevelError
    LevelFatal 
    LevelOff
)


// Return a human-friendly string for the severity level
func (l Level) String() string {
    switch l {
    case LevelInfo:
        return "INFO"
    case LevelError:
        return "ERROR"
    case LevelFatal:
        return "FATAL"
    default:
        return ""
    }
}

// Define a custom logger type. This holds the output destinations that the log
// entries will be written to, the minimum severity level that log entries will
// be written for, plus a mutex for coordinating the writes.
type Logger struct {
    out io.Writer
    minLevel Level
    mu sync.Mutex
}

// Return a new logger instance which writes log entries at or above a minimum 
// severity level to a specfic output destination.
func New(out io.Writer, minLevel Level) *Logger {
    return &Logger{
        out: out,
        minLevel: minLevel,
    }
}


// Declare some helper methods for writing log entries at the different level.
// Notice that these all accept a map as the second parameter which
// can contain any arbitrary 'properties' that you want to appear in the log entry
func (l *Logger) PrintInfo(message string, properties map[string]string) {
    l.print(LevelInfo, message, properties)
}

func (l *Logger) PrintError(err error, properties map[string]string) {
    l.print(LevelError, err.Error(), properties)
}

func (l *Logger) PrintFatal(err error, properties map[string]string) {
    l.print(LevelFatal, err.Error(), properties)
    os.Exit(1)  //For entries at the FATAL level, we also terminate the application
}

// Print is an internal method for writing the log entry
func (l *Logger) print(level Level, message string, properties map[string]string) (int, error) {
    // If the security level of the log entry is below the minimum severity for the logger,
    // then return with no further actions.
    if level < l.minLevel {
        return 0, nil
    }

    // Declare an anonymous struct holding the data for the log entry
    aux := struct {
        Level string `json:"level"`
        Time string `json:"time"`
        Message string `json:"message"`
        Properties map[string]string `json:"properties,omitempty"`
        Trace string `json:"trace,omitempty"`
    }{
        Level: level.String(),
        Time: time.Now().UTC().Format(time.RFC3339),
        Message: message,
        Properties: properties,
    }

    // Include a stack trace for entries at the error and FATAL levels.
    if level >= LevelError {
        aux.Trace = string(debug.Stack())
    }

    // Declare a line variable for holding the actual log entry text
    var line []byte

    // Marshal the anonymous struct to JSON and store it in the line variable.
    // If there was a problem creating the JSON, set the contents of the log
    // entry to be that plain-text error message instead 
    line, err := json.Marshal(aux)
    if err != nil {
        line = []byte(LevelError.String() + ": unable to marshal log message:" + err.Error())
    }

    // Lock the mutex so that no two writes to the output destination cannot happen
    // concurrently. If we don't do this, it's possible that the text for two or more
    // log entries will be intermingled in the output
    l.mu.Lock()
    defer l.mu.Unlock()

    // Write the log entry followed by a newline
    return l.out.Write(append(line, '\n'))
}


// We also implement a Write() method on our Logger type so that it satisfies the 
// io.Writer interface. This writes a log entry at the ERROR level with no additional
// properties.
func (l *Logger) Write(message []byte) (n int, err error) {
    return l.print(LevelError, string(message), nil)
}













