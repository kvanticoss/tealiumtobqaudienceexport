package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/voi-oss/svc"
	"github.com/kvanticoss/tealiumtobqaudienceexport/internal/httprouter"
	"github.com/kvanticoss/tealiumtobqaudienceexport/pkg/httpserver"
)

var (
	// Version the current code instance; usually commit hash
	Version    = "SNAPSHOT"
	Name       = "tealiumtobqaudienceexport"
	listenPort int

	projectID               string
	datasetID               string
	tableID                 string
	requiredAPIKey          string
	bigQueryDefaultLocation string = "EU"
)

func loadFlags() {
	flag.IntVar(&listenPort, "port", GetEnvWithDefaultInt("PORT", 8080), "server listen address")
	flag.StringVar(&projectID, "project", GetEnvWithDefaultString("PROJECT_ID", ""), "bigquery project id to use")
	flag.StringVar(&datasetID, "dataset", GetEnvWithDefaultString("DATASET", ""), "bigquery dataset to use")
	flag.StringVar(&tableID, "table", GetEnvWithDefaultString("TABLE", ""), "bigquery table to use")
	flag.StringVar(&requiredAPIKey, "apikey", GetEnvWithDefaultString("API_KEY", ""), "key to use in API requests")
	flag.StringVar(&bigQueryDefaultLocation, "location", GetEnvWithDefaultString("LOCATION", "EU"), "bigquery processing location")

	flag.Parse()
}

func main() {
	ctx := context.Background()

	env, _ := os.LookupEnv("GOOGLE_APPLICATION_CREDENTIALS")
	fmt.Printf("Environment:%v", env)

	loadFlags()

	s, err := svc.New(Name, Version)
	svc.MustInit(s, err)

	inserter, err := getTableInserter(ctx, projectID, datasetID, tableID)
	svc.MustInit(s, err)

	if requiredAPIKey == "" {
		panic("BQ Exporter must be started with an API key")
	}

	router := httprouter.New(inserter, map[string]string{"tealium_export": requiredAPIKey})

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
