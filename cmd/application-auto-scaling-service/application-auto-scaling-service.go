package main

import (
	"flag"
	"fmt"
	"os"

	"nanto.io/application-auto-scaling-service/cmd/application-auto-scaling-service/app"
)

const defaultConfPath = "/opt/cloud/application-auto-scaling-service/conf/application-auto-scaling-service.conf"

func main() {
	configFile := flag.String("config-file", defaultConfPath, "Service conf file")
	flag.Parse()

	if err := app.Run(*configFile); err != nil {
		fmt.Fprintf(os.Stderr, "app.Run err: %+v\n", err)
		os.Exit(1)
	}
}
