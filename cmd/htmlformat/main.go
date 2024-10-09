package main

import (
	"flag"
	"log"
	"os"

	"github.com/asartalo/formathtml"
)

var parseDocumentFlag = flag.Bool("document", false, "Set to true to parse a whole document")

func main() {
	flag.Parse()

	var err error
	if *parseDocumentFlag {
		err = formathtml.Document(os.Stdout, os.Stdin)
	} else {
		err = formathtml.Fragment(os.Stdout, os.Stdin)
	}
	if err != nil {
		log.Fatalf("failed to format: %v", err)
	}
}
