package model

import (
	"database/sql"
	"net/url"
)

type CustomVar struct {
	Raw  map[string]string
	JSON map[string]map[string]any
	LIST map[string][]any
}

type RequestContext struct {
	ClientIp        string
	Method          string
	Host            string
	Path            string
	Root            string
	ContentType     string
	ContentSource   string
	ServerVersion   string
	ServerError     map[string]string
	Query           url.Values
	Headers         map[string]string
	CustomVar       CustomVar
	ENV             map[string]string
	LocalVar        map[string]string
	SqlConn         *sql.DB
	GzipCompression bool
}
