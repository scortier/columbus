package postgres

import "errors"

var (
	errNilDBClient = errors.New("db client is nil")
)