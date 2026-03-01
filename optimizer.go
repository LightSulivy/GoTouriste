package main

import (
	"fmt"
	"math/rand"
	"time"
)

// Récupère les IDs des sites qu'on n'a pas encore réussi à glisser dans le trajet
func getUnvisited(inst *Instance, sol *Solution) []int {
	visited := make(map[int]bool)
	for _, day := range sol.Days {
		for _, step := range day.Steps {
			if inst.Points[step.PointID].Type == TypeSite {
				visited[step.PointID] = true
			}
		}
	}

	var res []int
	for _, id := range inst.SiteIDs {
		if !visited[id] {
			res = append(res, id)
		}
	}
	return res
}

// evalDay est une fonction hyper critique : elle vérifie si on peut refaire le trajet
// d'une journée précise avec un nouvel ordre de points, sans taper dans les limites
// de fermeture des sites ou dans le budget distance max.
func evalDay(inst *Instance, dayPoints []int) (bool, float64, []Step) {
	dist := 0.0
	t := 0.0
	steps := make([]Step, len(dayPoints))

	for i, pID := range dayPoints {
		pt := inst.Points[pID]
		steps[i].PointID = pID

		// Trajet depuis le point d'avant
		if i > 0 {
			prevPt := inst.Points[dayPoints[i-1]]
			d := inst.DistMatrix[prevPt.ID][pt.ID]
			dist += d
			t += d
			steps[i].DistFromPrev = d
		}

		steps[i].Arrival = t

		if pt.Type == TypeSite {
			wait := 0.0
			if t < pt.OpenTime {
				wait = pt.OpenTime - t // on est arrivé trop tôt, on poireaute
			}
			steps[i].Wait = wait
			startVisit := t + wait

			if startVisit > pt.CloseTime {
				// Raté, c'est fermé quand on arrive
				return false, 0, nil
			}
			t = startVisit + pt.ServiceTime
		}
		steps[i].Departure = t
	}

	// est-ce qu'on respecte le budget max en distance de la journée ?
	if dist > inst.MaxDist {
		return false, 0, nil
	}

	// Tout rentre, c'est valide
	return true, dist, steps
}

// LocalSearch : Métaheuristique simple (Type Hill Climbing / Recherche Locale)
// On va boucler et faire des petits mouvements (Insert, Swap, Relocate)
// jusqu'à que le chrono indique qu'il faut rendre la copie.
func LocalSearch(sol *Solution, maxDuration time.Duration) *Solution {
	bestSol := sol.Clone()
	bestSol.EvaluateScore()
	
	start := time.Now()
	iterations := 0
	
	fmt.Println("Recherche locale (métaheuristique) en cours...")

	for time.Since(start) < maxDuration {
		iterations++
		
		// On part toujours de la meilleure solution trouvée
		currentSol := bestSol.Clone()
		
		nbDays := len(currentSol.Days)
		if nbDays == 0 {
			break
		}

		dayIdx := rand.Intn(nbDays)
		day := &currentSol.Days[dayIdx]
		
		unvisited := getUnvisited(currentSol.Instance, currentSol)
		
		// Choix au hasard d'un mouvement : 0 (Insertion), 1 (Swap), 2 (Relocate intra-jour)
		moveType := rand.Intn(3)
		
		if moveType == 0 && len(unvisited) > 0 { 
			// INSERT : On essaye de glisser un site en plus pour gratter du score
			u := unvisited[rand.Intn(len(unvisited))]
			pts := make([]int, len(day.Steps))
			for i, s := range day.Steps { pts[i] = s.PointID }
			
			// On ne touche pas aux hôtels (index 0 et fin)
			if len(pts) >= 2 {
				pos := 1 + rand.Intn(len(pts)-1)
				
				newPts := make([]int, 0, len(pts)+1)
				newPts = append(newPts, pts[:pos]...)
				newPts = append(newPts, u)
				newPts = append(newPts, pts[pos:]...)
				
				valid, dist, newSteps := evalDay(currentSol.Instance, newPts)
				if valid {
					currentSol.Days[dayIdx].Steps = newSteps
					currentSol.Days[dayIdx].DistTotal = dist
					currentSol.EvaluateScore()
					
					// On prend si le score est plus grand, OU si on a raccourci le chemin à score égal
					if currentSol.TotalScore > bestSol.TotalScore {
						bestSol = currentSol
					} else if currentSol.TotalScore == bestSol.TotalScore && currentSol.TotalDist < bestSol.TotalDist {
						bestSol = currentSol
					}
				}
			}

		} else if moveType == 1 && len(unvisited) > 0 { 
			// SWAP : On vire un site visité et on en met un pas visité à la place
			if len(day.Steps) > 2 {
				// on s'assure de piocher une position qui n'est pas un hôtel
				pos := 1 + rand.Intn(len(day.Steps)-2)
				u := unvisited[rand.Intn(len(unvisited))]
				
				pts := make([]int, len(day.Steps))
				for i, s := range day.Steps { pts[i] = s.PointID }
				
				pts[pos] = u
				
				valid, dist, newSteps := evalDay(currentSol.Instance, pts)
				if valid {
					currentSol.Days[dayIdx].Steps = newSteps
					currentSol.Days[dayIdx].DistTotal = dist
					currentSol.EvaluateScore()
					
					if currentSol.TotalScore > bestSol.TotalScore || (currentSol.TotalScore == bestSol.TotalScore && currentSol.TotalDist < bestSol.TotalDist) {
						bestSol = currentSol
					}
				}
			}

		} else { 
			// RELOCATE : On décale un site pour le mettre un peu plus loin dans la même journée
			// Le but c'est juste de décroiser les chemins et d'économiser de la distance magiquement
			if len(day.Steps) > 3 {
				pos1 := 1 + rand.Intn(len(day.Steps)-2)
				pos2 := 1 + rand.Intn(len(day.Steps)-2)
				
				if pos1 != pos2 {
					pts := make([]int, len(day.Steps))
					for i, s := range day.Steps { pts[i] = s.PointID }
					
					// Extraction du point
					val := pts[pos1]
					pts = append(pts[:pos1], pts[pos1+1:]...) 
					
					// Ajustement de l'index suite au décalage
					if pos2 > pos1 { pos2-- }
					
					// Re-insertion
					newPts := make([]int, 0, len(pts)+1)
					newPts = append(newPts, pts[:pos2]...)
					newPts = append(newPts, val)
					newPts = append(newPts, pts[pos2:]...)
					
					valid, dist, newSteps := evalDay(currentSol.Instance, newPts)
					if valid {
						currentSol.Days[dayIdx].Steps = newSteps
						currentSol.Days[dayIdx].DistTotal = dist
						currentSol.EvaluateScore()
						
						// Le score sera le même mais la distance réduite
						if currentSol.TotalDist < bestSol.TotalDist {
							bestSol = currentSol
						}
					}
				}
			}
		}
	}

	fmt.Printf(">> Métaheuristique arrêtée. %d mouvements locaux testés.\n", iterations)
	return bestSol
}
