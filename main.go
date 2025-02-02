package main

import (
	"github.com/jdambly/kadmin/cmd"
	"log"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatalf("Error executing kadmin: %v", err)
	}
}
