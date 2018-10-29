package main

import (
	"net/http"
	"os"
	"strings"

	"github.com/apex/gateway"
	"github.com/apex/log"
	"github.com/julienschmidt/httprouter"
	datadog "github.com/matthewmueller/go-datadog"
)

func main() {
	key := os.Getenv("DATADOG_TOKEN")
	dd, err := datadog.Dial(&datadog.Config{APIKey: key})
	if err != nil {
		panic(err)
	}
	defer dd.Close()

	log := &Log{
		Interface: &log.Logger{
			Level:   log.InfoLevel,
			Handler: dd,
		},
		flush: dd.Flush,
	}

	router := httprouter.New()
	api := &API{router, log}
	router.HandlerFunc("GET", "/test", api.hello)
	log.Info("api is listening")

	if err := gateway.ListenAndServe(":3000", api); err != nil {
		log.WithError(err).Fatal("server died")
	}
}

// API struct
type API struct {
	router http.Handler
	log    log.Interface
}

var _ http.Handler = (*API)(nil)

func (a *API) hello(w http.ResponseWriter, r *http.Request) {
	headers := map[string]string{}
	for k, v := range r.Header {
		headers[k] = strings.Join(v, ", ")
	}

	flusher, ok := a.log.(Flusher)
	if !ok {
		panic("not a flusher")
	}
	defer flusher.Flush()

	a.log.WithFields(log.Fields{
		"host":    r.Host,
		"ip":      r.RemoteAddr,
		"headers": headers,
		"ua":      r.UserAgent(),
	}).Infof("%s %s %s %s", r.Proto, r.Method, r.URL.RawPath, r.URL.RawQuery)

	w.Write([]byte("hi datadog support!"))
}

func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.router.ServeHTTP(w, r)
}

// Flusher logs before exiting
type Flusher interface {
	Flush()
}

// Log wrapper to add flush
type Log struct {
	log.Interface
	flush func()
}

// Flush calls datadog's flush
func (l *Log) Flush() {
	l.flush()
}
