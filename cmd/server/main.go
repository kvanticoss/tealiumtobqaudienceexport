package main

import (
	"flag"
	"net"
	"strconv"

	"github.com/voi-oss/svc"
	"github.com/zatte/golang-template/internal/httprouter"
	"github.com/zatte/golang-template/pkg/httpserver"
)

var (
	// Version the current code instance; usually commit hash
	Version    = "SNAPSHOT"
	Name       = "golang-template"
	listenPort int
)

func loadFlags() {
	flag.IntVar(&listenPort, "port", 8080, "server listen address")
	flag.Parse()
}

func main() {
	loadFlags()

	s, err := svc.New(Name, Version)
	svc.MustInit(s, err)

	router := httprouter.New()

	options := append(
		httpserver.DefaultOptions,
		httpserver.WithListenAdr(
			net.JoinHostPort("", strconv.Itoa(listenPort)),
		),
	)

	s.AddWorker("http-server",
		httpserver.New(router, options...),
	)

	s.Run()
}
