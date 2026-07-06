package main

import (
	"fmt"
	"log"
	"os"

	"github.com/bidon-io/bidon-backend/internal/dspsimulator"
)

func main() {
	responseDir := os.Getenv("DSP_RESPONSE_DIR")
	if responseDir == "" {
		responseDir = "./internal/sdkapi/v2/apihandlers/testdata/auction/adikteev"
	}

	port := os.Getenv("DSP_SIMULATOR_PORT")
	if port == "" {
		port = "1326"
	}

	store, err := dspsimulator.NewResponseStore(responseDir)
	if err != nil {
		log.Fatalf("failed to load responses: %v", err)
	}

	svc := dspsimulator.NewService(store)
	srv := dspsimulator.NewServer(svc)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("DSP simulator listening on %s", addr)
	log.Fatal(srv.Start(addr))
}
