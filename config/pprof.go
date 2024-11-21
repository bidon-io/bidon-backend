package config

import (
	_ "net/http/pprof" // Import pprof to register its handlers
)

// StartPprof initializes the HTTP server for pprof profiling.
//func StartPprof(address string) {
//	go func() {
//		log.Printf("Starting pprof on %s", address)
//		if err := http.ListenAndServe(address, nil); err != nil {
//			log.Fatalf("Error starting pprof: %v", err)
//		}
//	}()
//}
