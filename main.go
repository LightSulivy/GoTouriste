package main

import (
	"fmt"
	"log"
	"path/filepath"
)

func main() {
	// 1. Charger l'instance
	instancePath := filepath.Join("instances", "Inst1.txt")
	instance, err := LoadInstance(instancePath)
	if err != nil {
		log.Fatalf("Erreur chargement instance %s: %v", instancePath, err)
	}

	fmt.Printf("Instance chargée : %d sites, %d hôtels, Budget : %.2f\n", len(instance.Points)-len(instance.HotelIDs), len(instance.HotelIDs), instance.MaxDist)

	// 2. Résoudre avec l'algo Glouton
	solution := SolveGreedy(instance)
	solution.EvaluateScore()

	fmt.Printf("Solution Gloutonne trouvée :\n")
	fmt.Printf("  Score Total : %.2f\n", solution.TotalScore)
	fmt.Printf("  Distance Totale : %.2f\n", solution.TotalDist)

	// 3. Exporter la solution
	outputPath := "Inst1.sol"
	err = WriteSolution(solution, outputPath)
	if err != nil {
		log.Fatalf("Erreur lors de l'écriture de la solution : %v", err)
	}
	fmt.Printf("Solution écrite dans %s\n", outputPath)
}
