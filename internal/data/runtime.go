package data

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrInvalidRuntimeFormat = errors.New("invalid runtime format")

type Runtime int32

func (r *Runtime) UnmarshalJSON(jsonvalue []byte) error {
    
    unquotedJSONvalue, err := strconv.Unquote(string(jsonvalue))
    if err != nil {
        return ErrInvalidRuntimeFormat
    }

    parts := strings.Split(unquotedJSONvalue, " ")

    if len(parts) != 2 || parts[1] != "mins" {
        return ErrInvalidRuntimeFormat
    }

    i, err := strconv.ParseInt(parts[0], 10, 32)
    if err != nil {
        return ErrInvalidRuntimeFormat
    }

    *r = Runtime(i)

    return nil
    
}

func(r Runtime) MarshalJSON() ([]byte, error) {
    jsonValue := fmt.Sprintf("%d mins", r)

    quotedJSONValue := strconv.Quote(jsonValue)

    return []byte(quotedJSONValue), nil
}
