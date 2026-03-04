package main

import (
	"math"
	"math/rand"
	"sort"
)

<<<<<<< HEAD
type candidate struct {
	id        int
	ratio     float64
	arrival   float64
	wait      float64
	departure float64
	dist      float64
}

// SolveGreedy construit une solution déterministe (meilleur ratio à chaque étape).
=======
// SolveGreedy construit une solution via une approche gloutonne
>>>>>>> 8d3553d17528eb8142d1bd8548290aceb867257b
func SolveGreedy(inst *Instance) *Solution {
	return solveGreedyInternal(inst, 1, nil, 0)
}

// SolveGreedyRandomized construit une solution randomisée (pioche parmi les k meilleurs).
func SolveGreedyRandomized(inst *Instance, rclSize int, rng *rand.Rand, ratioMode int) *Solution {
	return solveGreedyInternal(inst, rclSize, rng, ratioMode)
}

// rclSize=1 → déterministe, rclSize>1 → randomisé (pioche parmi les top-k)
func solveGreedyInternal(inst *Instance, rclSize int, rng *rand.Rand, ratioMode int) *Solution {
	sol := &Solution{
		Instance: inst,
		Days:     make([]DayTour, 0),
	}

	visited := make([]bool, len(inst.Points))
	currentLocationID := inst.StartHotelID

	for d := 0; d < inst.NbDays; d++ {
		steps := []Step{
			{PointID: currentLocationID, Arrival: 0, Wait: 0, Departure: 0, DistFromPrev: 0},
		}

		for {
			bestRatio := -1.0
			var bestCand candidate
			var candidates []candidate

			for i, p := range inst.Points {
				if visited[i] || p.Type == TypeHotel {
					continue
				}

				dist := inst.DistMatrix[currentLocationID][i]
				arrival := steps[len(steps)-1].Departure + dist

				if arrival > p.CloseTime {
					continue
				}

				wait := 0.0
				if arrival < p.OpenTime {
					wait = p.OpenTime - arrival
				}
				departure := arrival + wait + p.ServiceTime

				canReturn := false
				if d == inst.NbDays-1 {
					distBack := inst.DistMatrix[i][inst.EndHotelID]
					if departure+distBack <= inst.DayMaxDist(d) {
						canReturn = true
					}
				} else {
					for _, hID := range inst.HotelIDs {
						distBack := inst.DistMatrix[i][hID]
						if departure+distBack <= inst.DayMaxDist(d) {
							canReturn = true
							break
						}
					}
				}

				if !canReturn {
					continue
				}

				cost := dist + wait
				if cost < 0.001 {
					cost = 0.001
				}

				var ratio float64
				switch ratioMode {
				case 1:
					// Mode urgence : priorise les sites qui ferment bientôt
					window := p.CloseTime - arrival
					if window < 0.001 {
						window = 0.001
					}
					ratio = p.Score / window
				case 2:
					// Mode score pur : ignore la distance
					ratio = p.Score
				case 3:
					// Mode proximité : nearest neighbor
					if dist < 0.001 {
						ratio = 10000
					} else {
						ratio = 1.0 / dist
					}
				default:
					// Mode 0 : ratio classique Score/Coût
					ratio = p.Score / cost
				}

				c := candidate{id: i, ratio: ratio, arrival: arrival, wait: wait, departure: departure, dist: dist}

				if rclSize <= 1 {
					// Mode déterministe : on garde juste le meilleur
					if ratio > bestRatio {
						bestRatio = ratio
						bestCand = c
					}
				} else {
					candidates = append(candidates, c)
				}
			}

			// Sélection
			var chosen candidate
			found := false
			if rclSize <= 1 {
				if bestRatio >= 0 {
					chosen = bestCand
					found = true
				}
			} else if len(candidates) > 0 {
				sort.Slice(candidates, func(a, b int) bool {
					return candidates[a].ratio > candidates[b].ratio
				})
				k := rclSize
				if k > len(candidates) {
					k = len(candidates)
				}
				chosen = candidates[rng.Intn(k)]
				found = true
			}

			if !found {
				break
			}

			steps = append(steps, Step{
				PointID:      chosen.id,
				Arrival:      chosen.arrival,
				Wait:         chosen.wait,
				Departure:    chosen.departure,
				DistFromPrev: chosen.dist,
			})
			visited[chosen.id] = true
			currentLocationID = chosen.id
		}

		// Fin de journée : choix hôtel
		var endHotelID int
		if d == inst.NbDays-1 {
			endHotelID = inst.EndHotelID
		} else {
			bestH := -1
			minDistH := math.MaxFloat64
			lastStep := steps[len(steps)-1]

			for _, hID := range inst.HotelIDs {
				dist := inst.DistMatrix[lastStep.PointID][hID]
				if lastStep.Departure+dist <= inst.DayMaxDist(d) {
					if dist < minDistH {
						minDistH = dist
						bestH = hID
					}
				}
			}
			if bestH == -1 {
				bestH = inst.EndHotelID
			}
			endHotelID = bestH
		}

		lastStep := steps[len(steps)-1]
		distToEnd := inst.DistMatrix[lastStep.PointID][endHotelID]
		steps = append(steps, Step{
			PointID:      endHotelID,
			Arrival:      lastStep.Departure + distToEnd,
			Wait:         0,
			Departure:    lastStep.Departure + distToEnd,
			DistFromPrev: distToEnd,
		})

		dayTour := DayTour{Steps: steps}
		for _, s := range steps {
			dayTour.DistTotal += s.DistFromPrev
		}
		dayTour.TimeTotal = steps[len(steps)-1].Arrival
		sol.Days = append(sol.Days, dayTour)

		currentLocationID = endHotelID
	}

	sol.EvaluateScore()
	return sol
}
