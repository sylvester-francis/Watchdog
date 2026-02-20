package main

import (
	"context"
	"log"

	"github.com/sylvester-francis/watchdog/engine"
)

func main() {
	ctx := context.Background()

	eng, err := engine.New(ctx)
	if err != nil {
		log.Fatal(err)
	}

	if err := eng.Init(ctx); err != nil {
		log.Fatal(err)
	}

	if err := eng.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
