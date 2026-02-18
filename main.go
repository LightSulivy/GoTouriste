package main

import (
	"fmt"
	"log"
	"path/filepath"
)

func main() {
	// Exemple avec Inst1.txt
	instancePath := filepath.Join("instances", "Inst1.txt")

	instance, err := LoadInstance(instancePath)
	if err != nil {
		log.Fatalf("Erreur chargement instance %s: %v", instancePath, err)
	}

	// Calcul des stats pour l'affichage
	nbSites := 0
	nbHotels := 0
	for _, p := range instance.Points {
		if p.Type == TypeSite {
			nbSites++
		} else if p.Type == TypeHotel {
			nbHotels++
		}
	}

	fmt.Printf("Instance chargée : %d sites, %d hôtels, Budget : %.2f\n", nbSites, nbHotels, instance.MaxDist)
}
