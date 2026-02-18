package main

import (
	"math"
)

// SolveGreedy construit une solution via une approche gloutonne.
func SolveGreedy(inst *Instance) *Solution {
	sol := &Solution{
		Instance: inst,
		Days:     make([]DayTour, 0),
	}

	visited := make(map[int]bool)
	currentLocationID := inst.StartHotelID // Départ du 1er jour

	for d := 0; d < inst.NbDays; d++ {
		dayTour := DayTour{
			Steps: make([]Step, 0),
		}

		// Ajout du point de départ
		steps := []Step{
			{
				PointID:      currentLocationID,
				Arrival:      0,
				Wait:         0,
				Departure:    0,
				DistFromPrev: 0,
			},
		}

		// Boucle gloutonne
		for {
			bestCandidateID := -1
			bestScoreRatio := -1.0
			var bestArrival, bestWait, bestDeparture, bestDist float64

			// Chercher le meilleur candidat non visité
			for i, p := range inst.Points {
				if visited[i] {
					continue
				}
				// On ne s'arrête pas aux hôtels en cours de journée
				if p.Type == TypeHotel {
					continue
				}

				dist := inst.DistMatrix[currentLocationID][i]
				arrival := steps[len(steps)-1].Departure + dist

				// Vérif fenêtre de temps
				if arrival > p.CloseTime {
					continue
				}

				wait := 0.0
				if arrival < p.OpenTime {
					wait = p.OpenTime - arrival
				}
				departure := arrival + wait + p.ServiceTime

				// Vérif retour hôtel (Fin ou Nimp lequel selon le jour)
				canReturn := false
				if d == inst.NbDays-1 {
					// Dernier jour : impératif retour EndHotelID
					distBack := inst.DistMatrix[i][inst.EndHotelID]
					if departure+distBack <= inst.MaxDist {
						canReturn = true
					}
				} else {
					// Autre jour : peut-on atteindre un hôtel quelconque ?
					for _, hID := range inst.HotelIDs {
						distBack := inst.DistMatrix[i][hID]
						if departure+distBack <= inst.MaxDist {
							canReturn = true
							break
						}
					}
				}

				if !canReturn {
					continue
				}

				// Heuristique : Score / Coût (Distance + Attente)
				cost := dist + wait
				if cost < 0.001 {
					cost = 0.001
				}
				ratio := p.Score / cost

				if ratio > bestScoreRatio {
					bestScoreRatio = ratio
					bestCandidateID = i
					bestArrival = arrival
					bestWait = wait
					bestDeparture = departure
					bestDist = dist
				}
			}

			if bestCandidateID != -1 {
				// Ajout étape
				steps = append(steps, Step{
					PointID:      bestCandidateID,
					Arrival:      bestArrival,
					Wait:         bestWait,
					Departure:    bestDeparture,
					DistFromPrev: bestDist,
				})
				visited[bestCandidateID] = true
				currentLocationID = bestCandidateID
			} else {
				// Plus de candidat valide pour ce jour
				break
			}
		}

		// Fin de journée : choix hôtel
		var endHotelID int
		if d == inst.NbDays-1 {
			endHotelID = inst.EndHotelID
		} else {
			// Trouver l'hôtel accessible le plus proche
			bestH := -1
			minDistH := math.MaxFloat64

			lastStep := steps[len(steps)-1]

			for _, hID := range inst.HotelIDs {
				dist := inst.DistMatrix[lastStep.PointID][hID]
				if lastStep.Departure+dist <= inst.MaxDist {
					if dist < minDistH {
						minDistH = dist
						bestH = hID
					}
				}
			}
			// Fallback (ne devrait pas arriver si algo OK)
			if bestH == -1 {
				bestH = inst.EndHotelID
			}
			endHotelID = bestH
		}

		// Ajout trajet vers hôtel fin
		lastStep := steps[len(steps)-1]
		distToEnd := inst.DistMatrix[lastStep.PointID][endHotelID]
		steps = append(steps, Step{
			PointID:      endHotelID,
			Arrival:      lastStep.Departure + distToEnd,
			Wait:         0,
			Departure:    lastStep.Departure + distToEnd,
			DistFromPrev: distToEnd,
		})

		// Finalisation jour
		dayTour.Steps = steps
		dayTour.DistTotal = 0
		for _, s := range steps {
			dayTour.DistTotal += s.DistFromPrev
		}
		dayTour.TimeTotal = steps[len(steps)-1].Arrival

		sol.Days = append(sol.Days, dayTour)

		// Le lendemain démarre ici
		currentLocationID = endHotelID
	}

	sol.EvaluateScore()
	return sol
}
