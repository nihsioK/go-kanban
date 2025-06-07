package app

import (
	"log"
	"os"
)

func loadSchemas() map[string]string {
	files := map[string]string{
		"user":    "schemas/user.json",
		"project": "schemas/project.json",
	}

	schemas := make(map[string]string)

	for name, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			log.Fatalf("Failed to load schema %s: %v", name, err)
		}
		schemas[name] = string(data)
	}

	return schemas
}
