package main

import (
	"fmt"
	"log"
	"path/filepath"
	"time"
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

	fmt.Printf("\n--- Solution Gloutonne initiale ---\n")
	fmt.Printf("  Score : %.2f\n", solution.TotalScore)
	fmt.Printf("  Distance : %.2f\n", solution.TotalDist)

	// On se donne 5 secondes pour améliorer la solution (pour tester vite, à monter à 120s pour la compet)
	fmt.Println("\nLancement de la phase d'optimisation...")
	optSolution := LocalSearch(solution, 5*time.Second)

	fmt.Printf("\n--- Solution après Optimisation ---\n")
	fmt.Printf("  Score : %.2f\n", optSolution.TotalScore)
	fmt.Printf("  Distance : %.2f\n", optSolution.TotalDist)

	// Juge : Validation officielle finale avant de rendre la copie
	valid, errValid := EvaluateSolution(optSolution)
	if !valid {
		log.Fatalf("Alerte: La solution optimisée est invalide !! Erreur : %v", errValid)
	} else {
		fmt.Println("Vérification OK : La solution respecte toutes les règles du concours.")
	}

	// 3. Exporter la solution finale
	outputPath := "Inst1.sol"
	err = WriteSolution(optSolution, outputPath)
	if err != nil {
		log.Fatalf("Erreur lors de l'écriture de la solution : %v", err)
	}
	fmt.Printf("\nSolution finale écrite dans %s\n", outputPath)
}
