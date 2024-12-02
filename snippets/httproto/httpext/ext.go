package httpext

import "net/http"

type HandleFunc func(http.ResponseWriter, *http.Request)
