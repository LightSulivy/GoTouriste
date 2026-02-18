package main

import (
	"fmt"
	"os"
	"strings"
)

// WriteSolution écrit la solution dans un fichier (1 ligne par jour avec IDs)
func WriteSolution(sol *Solution, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("impossible de créer le fichier solution : %v", err)
	}
	defer file.Close()

	for _, day := range sol.Days {
		var ids []string
		for _, step := range day.Steps {
			// Format: ID des points
			ids = append(ids, fmt.Sprintf("%d", step.PointID))
		}
		line := strings.Join(ids, " ")
		_, err := fmt.Fprintln(file, line)
		if err != nil {
			return err
		}
	}

	return nil
}
