package main

import (
	"fmt"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/util/rand"

	"nanto.io/application-auto-scaling-service/cmd/application-auto-scaling-service/app"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "app.Run err: %+v\n", err)
		os.Exit(1)
	}
}
