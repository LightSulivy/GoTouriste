package main

import (
	"fmt"
	"os"
	"strings"
)

// WriteSolution écrit la solution au format officiel du concours.
// Ligne 1 : score total
// Lignes suivantes : HotelDepart Site1 Site2 ... HotelFin (une ligne par jour)
func WriteSolution(sol *Solution, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("impossible de créer le fichier solution : %v", err)
	}
	defer file.Close()

	// Première ligne : valeur de la solution
	fmt.Fprintf(file, "%.0f\n", sol.TotalScore)

	for _, day := range sol.Days {
		var ids []string
		for _, step := range day.Steps {
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
