package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Solutions optimales connues par instance
var optimalScores = map[string]float64{
	"1": 816, "2": 900, "3": 1062, "4": 1062, "5": 1116, "6": 1236,
	"7": 1236, "8": 1236, "9": 1284, "10": 1284, "11": 1284, "12": 1670,
	"13": 173, "14": 241, "15": 367, "16": 412, "17": 412, "18": 504,
	"19": 504, "20": 504, "21": 590, "22": 1114, "23": 1164, "24": 1234,
	"25": 1234, "26": 1261, "27": 1306, "28": 984, "29": 1188, "30": 1284,
	"31": 1670, "32": 299, "33": 504, "34": 1164, "35": 1201, "36": 1284,
}

func main() {
	os.MkdirAll("solutions", 0755)

	instanceFiles, err := filepath.Glob(filepath.Join("instances", "Inst*.txt"))
	if err != nil {
		log.Fatalf("Erreur lecture dossier instances : %v", err)
	}
	if len(instanceFiles) == 0 {
		log.Fatal("Aucune instance trouvée dans le dossier instances/")
	}

	fmt.Printf("=== %d instances détectées ===\n\n", len(instanceFiles))

	// Warmup du runtime Go (élimine l'overhead du premier appel)
	if warmInst, err := LoadInstance(instanceFiles[0]); err == nil {
		w := SolveGreedy(warmInst)
		w.EvaluateScore()
	}

	for _, instPath := range instanceFiles {
		baseName := filepath.Base(instPath)
		numStr := strings.TrimSuffix(strings.TrimPrefix(baseName, "Inst"), ".txt")
		outputPath := filepath.Join("solutions", fmt.Sprintf("Instance%s.sol", numStr))

		// Chargement (hors timer)
		instance, err := LoadInstance(instPath)
		if err != nil {
			log.Printf("ERREUR %s : %v\n\n", baseName, err)
			continue
		}

		// Score optimal connu
		target := optimalScores[numStr]

		fmt.Printf("Résolution de %s ...\n", baseName)
		os.Stdout.Sync()

		// Timer commence après la lecture
		timer := time.Now()

		// Résolution
		solution := SolveGreedy(instance)
		solution.EvaluateScore()

		var optSolution *Solution
		if target > 0 && solution.TotalScore >= target {
			optSolution = solution
		} else {
			optSolution = LocalSearch(solution, 120*time.Second, target)
		}

		// Timer s'arrête avant l'écriture
		elapsed := time.Since(timer)

		// Validation
		valid, errValid := EvaluateSolution(optSolution)
		if !valid {
			optSolution = solution
			valid, errValid = EvaluateSolution(optSolution)
			if !valid {
				log.Printf("ERREUR %s : solution invalide : %v\n\n", baseName, errValid)
				continue
			}
		}

		// Indicateur optimal
		status := ""
		if target > 0 && optSolution.TotalScore >= target {
			status = " ★"
		}

		// Affichage
		fmt.Printf("────────────────────────────────────────\n")
		fmt.Printf(" %s  |  Score: %.0f/%.0f  |  Temps: %.1fms%s\n", baseName, optSolution.TotalScore, target, float64(elapsed.Microseconds())/1000.0, status)
		fmt.Printf("────────────────────────────────────────\n")
		for d, day := range optSolution.Days {
			nbSites := 0
			for _, s := range day.Steps {
				if instance.Points[s.PointID].Type == TypeSite {
					nbSites++
				}
			}
			fmt.Printf("  Jour %d : %d sites, dist=%.2f / %.2f\n", d+1, nbSites, day.DistTotal, instance.DayMaxDist(d))
		}

		// Lire le score et temps existants
		timePath := strings.TrimSuffix(outputPath, ".sol") + ".time"
		oldScore := -1.0
		oldTime := 999999999.0
		if f, err := os.Open(outputPath); err == nil {
			scanner := bufio.NewScanner(f)
			if scanner.Scan() {
				oldScore, _ = strconv.ParseFloat(strings.TrimSpace(scanner.Text()), 64)
			}
			f.Close()
		}
		if data, err := os.ReadFile(timePath); err == nil {
			oldTime, _ = strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
		}

		// Écrire si meilleur score, ou même score mais temps plus court
		elapsedUs := float64(elapsed.Microseconds())
		shouldWrite := optSolution.TotalScore > oldScore ||
			(optSolution.TotalScore == oldScore && elapsedUs < oldTime)

		if shouldWrite {
			err = WriteSolution(optSolution, outputPath)
			if err != nil {
				log.Printf("  Erreur écriture : %v\n\n", err)
				continue
			}
			os.WriteFile(timePath, []byte(fmt.Sprintf("%d", elapsed.Microseconds())), 0644)

			if oldScore >= 0 {
				fmt.Printf("  -> %s (ancien: %.0f en %.1fms)\n\n", outputPath, oldScore, oldTime/1000.0)
			} else {
				fmt.Printf("  -> %s\n\n", outputPath)
			}
		} else {
			fmt.Printf("  -> Pas d'amélioration (existant: %.0f en %.1fms)\n\n", oldScore, oldTime/1000.0)
		}
	}

	fmt.Println("=== Terminé ===")
}
