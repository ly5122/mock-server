package main

import (
	"flag"
	"net/http"

	logger "log"
)

var (
	addr = flag.String("addr", ":80", "tcp address to listen to")
	maxSize = 1024 * 1024
)

func main() {
	flag.Parse()
	initLog()
	e := newEngine()
	e.init()
	err := http.ListenAndServe(*addr, e)
	if err != nil {
		logger.Print(err)
	}
}

func initLog() {
	logger.Print("\n>> Welcome to mock-server\n>> Any questions please contact ly5122@github.com")
}
