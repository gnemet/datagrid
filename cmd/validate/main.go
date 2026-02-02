package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/xeipuuv/gojsonschema"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: datagrid-validator <schema_path> <catalog_path1> [catalog_path2] ...")
		os.Exit(1)
	}

	schemaPath, err := filepath.Abs(os.Args[1])
	if err != nil {
		log.Fatalf("Invalid schema path: %v", err)
	}
	schemaLoader := gojsonschema.NewReferenceLoader("file://" + schemaPath)

	allValid := true
	for i := 2; i < len(os.Args); i++ {
		catalogPath, err := filepath.Abs(os.Args[i])
		if err != nil {
			fmt.Printf("❌ Invalid catalog path: %s\n", os.Args[i])
			allValid = false
			continue
		}

		documentLoader := gojsonschema.NewReferenceLoader("file://" + catalogPath)
		result, err := gojsonschema.Validate(schemaLoader, documentLoader)
		if err != nil {
			fmt.Printf("❌ Error validating %s: %v\n", filepath.Base(catalogPath), err)
			allValid = false
			continue
		}

		if result.Valid() {
			fmt.Printf("✅ %s is valid.\n", filepath.Base(catalogPath))
		} else {
			fmt.Printf("❌ %s is invalid!\n", filepath.Base(catalogPath))
			for _, desc := range result.Errors() {
				fmt.Printf("   - %s\n", desc)
			}
			allValid = false
		}
	}

	if !allValid {
		os.Exit(1)
	}
}
