package main

import (
	"fmt"
	"math/rand"
	"time"
)

type SearchState struct {
	sol       *Solution
	visited   []bool
	unvisited []int
}

func newSearchState(sol *Solution) *SearchState {
	inst := sol.Instance
	n := len(inst.Points)

	visited := make([]bool, n)
	for _, day := range sol.Days {
		for _, step := range day.Steps {
			if inst.Points[step.PointID].Type == TypeSite {
				visited[step.PointID] = true
			}
		}
	}

	var unvisited []int
	for _, id := range inst.SiteIDs {
		if !visited[id] {
			unvisited = append(unvisited, id)
		}
	}

	return &SearchState{
		sol:       sol,
		visited:   visited,
		unvisited: unvisited,
	}
}

func (ss *SearchState) markVisited(siteID int) {
	ss.visited[siteID] = true
	for i, id := range ss.unvisited {
		if id == siteID {
			ss.unvisited[i] = ss.unvisited[len(ss.unvisited)-1]
			ss.unvisited = ss.unvisited[:len(ss.unvisited)-1]
			break
		}
	}
}

func (ss *SearchState) markUnvisited(siteID int) {
	ss.visited[siteID] = false
	ss.unvisited = append(ss.unvisited, siteID)
}

// evalDay vérifie la faisabilité d'un trajet et retourne les Steps
func evalDay(inst *Instance, dayPoints []int) (bool, float64, []Step) {
	dist := 0.0
	t := 0.0
	maxDist := inst.MaxDist
	steps := make([]Step, len(dayPoints))

	for i, pID := range dayPoints {
		pt := inst.Points[pID]
		steps[i].PointID = pID

		if i > 0 {
			d := inst.DistMatrix[dayPoints[i-1]][pID]
			dist += d
			if dist > maxDist {
				return false, 0, nil
			}
			t += d
			steps[i].DistFromPrev = d
		}

		steps[i].Arrival = t

		if pt.Type == TypeSite {
			wait := 0.0
			if t < pt.OpenTime {
				wait = pt.OpenTime - t
			}
			steps[i].Wait = wait
			startVisit := t + wait

			if startVisit > pt.CloseTime {
				return false, 0, nil
			}
			t = startVisit + pt.ServiceTime
		}
		steps[i].Departure = t
	}

	return true, dist, steps
}

// evalDayFast vérifie la faisabilité sans construire les Steps
func evalDayFast(inst *Instance, dayPoints []int) (bool, float64) {
	dist := 0.0
	t := 0.0
	maxDist := inst.MaxDist

	for i, pID := range dayPoints {
		pt := inst.Points[pID]

		if i > 0 {
			d := inst.DistMatrix[dayPoints[i-1]][pID]
			dist += d
			if dist > maxDist {
				return false, 0
			}
			t += d
		}

		if pt.Type == TypeSite {
			wait := 0.0
			if t < pt.OpenTime {
				wait = pt.OpenTime - t
			}
			startVisit := t + wait
			if startVisit > pt.CloseTime {
				return false, 0
			}
			t = startVisit + pt.ServiceTime
		}
	}

	return true, dist
}

func extractDayPoints(day *DayTour, buf []int) []int {
	buf = buf[:0]
	for _, s := range day.Steps {
		buf = append(buf, s.PointID)
	}
	return buf
}

func scoreDelta(inst *Instance, addedID int, removedID int) float64 {
	delta := 0.0
	if addedID >= 0 {
		delta += inst.Points[addedID].Score
	}
	if removedID >= 0 {
		delta -= inst.Points[removedID].Score
	}
	return delta
}

// LocalSearch applique des mouvements locaux (Insert, Swap, Relocate, 2-opt)
// Arrêt au timeout ou si stagnation détectée
func LocalSearch(sol *Solution, maxDuration time.Duration) *Solution {
	bestSol := sol.Clone()
	bestSol.EvaluateScore()
	bestState := newSearchState(bestSol)

	start := time.Now()
	iterations := 0
	improvements := 0
	sinceLastImprove := 0

	// Seuil de stagnation adapté à la taille de l'instance
	nbPoints := len(sol.Instance.Points)
	maxStagnation := 50000
	if nbPoints > 80 {
		maxStagnation = 150000
	} else if nbPoints > 50 {
		maxStagnation = 100000
	}

	fmt.Println("Recherche locale en cours...")

	ptsBuf := make([]int, 0, 128)
	newPtsBuf := make([]int, 0, 128)

	for time.Since(start) < maxDuration {
		iterations++
		sinceLastImprove++

		if sinceLastImprove > maxStagnation {
			fmt.Printf(">> Convergence atteinte après %d itérations sans progrès.\n", maxStagnation)
			break
		}

		nbDays := len(bestSol.Days)
		if nbDays == 0 {
			break
		}

		dayIdx := rand.Intn(nbDays)
		day := &bestSol.Days[dayIdx]
		nbSteps := len(day.Steps)

		moveType := rand.Intn(4)

		if moveType == 0 && len(bestState.unvisited) > 0 && nbSteps >= 2 {
			// INSERT : ajouter un site non visité
			uIdx := rand.Intn(len(bestState.unvisited))
			u := bestState.unvisited[uIdx]
			pos := 1 + rand.Intn(nbSteps-1)

			ptsBuf = extractDayPoints(day, ptsBuf)
			newPtsBuf = newPtsBuf[:0]
			newPtsBuf = append(newPtsBuf, ptsBuf[:pos]...)
			newPtsBuf = append(newPtsBuf, u)
			newPtsBuf = append(newPtsBuf, ptsBuf[pos:]...)

			feasible, newDist := evalDayFast(sol.Instance, newPtsBuf)
			if feasible {
				_, _, newSteps := evalDay(sol.Instance, newPtsBuf)
				bestSol.Days[dayIdx].Steps = newSteps
				bestSol.Days[dayIdx].DistTotal = newDist
				bestSol.TotalScore += sol.Instance.Points[u].Score
				bestSol.TotalDist += newDist - day.DistTotal
				bestState.markVisited(u)
				improvements++
				sinceLastImprove = 0
			}

		} else if moveType == 1 && len(bestState.unvisited) > 0 && nbSteps > 2 {
			// SWAP : remplacer un site visité par un non-visité
			pos := 1 + rand.Intn(nbSteps-2)
			oldID := day.Steps[pos].PointID

			if sol.Instance.Points[oldID].Type != TypeSite {
				continue
			}

			uIdx := rand.Intn(len(bestState.unvisited))
			u := bestState.unvisited[uIdx]

			ptsBuf = extractDayPoints(day, ptsBuf)
			ptsBuf[pos] = u

			feasible, newDist := evalDayFast(sol.Instance, ptsBuf)
			if feasible {
				delta := scoreDelta(sol.Instance, u, oldID)
				if delta > 0 || (delta == 0 && newDist < day.DistTotal) {
					_, _, newSteps := evalDay(sol.Instance, ptsBuf)
					bestSol.Days[dayIdx].Steps = newSteps
					bestSol.Days[dayIdx].DistTotal = newDist
					bestSol.TotalScore += delta
					bestSol.TotalDist += newDist - day.DistTotal
					bestState.markVisited(u)
					bestState.markUnvisited(oldID)
					improvements++
					sinceLastImprove = 0
				}
			}

		} else if moveType == 2 && nbSteps > 3 {
			// RELOCATE : déplacer un site à une autre position dans la journée
			pos1 := 1 + rand.Intn(nbSteps-2)
			pos2 := 1 + rand.Intn(nbSteps-2)
			if pos1 == pos2 {
				continue
			}

			ptsBuf = extractDayPoints(day, ptsBuf)
			oldDist := day.DistTotal

			val := ptsBuf[pos1]
			copy(ptsBuf[pos1:], ptsBuf[pos1+1:])
			ptsBuf = ptsBuf[:len(ptsBuf)-1]

			if pos2 > pos1 {
				pos2--
			}

			newPtsBuf = newPtsBuf[:0]
			newPtsBuf = append(newPtsBuf, ptsBuf[:pos2]...)
			newPtsBuf = append(newPtsBuf, val)
			newPtsBuf = append(newPtsBuf, ptsBuf[pos2:]...)

			feasible, newDist := evalDayFast(sol.Instance, newPtsBuf)
			if feasible && newDist < oldDist {
				_, _, newSteps := evalDay(sol.Instance, newPtsBuf)
				bestSol.Days[dayIdx].Steps = newSteps
				bestSol.TotalDist += newDist - oldDist
				bestSol.Days[dayIdx].DistTotal = newDist
				improvements++
				sinceLastImprove = 0
			}

		} else if nbSteps > 3 {
			// 2-OPT : inverser un sous-segment pour décroiser le trajet
			a := 1 + rand.Intn(nbSteps-2)
			b := 1 + rand.Intn(nbSteps-2)
			if a == b {
				continue
			}
			if a > b {
				a, b = b, a
			}

			ptsBuf = extractDayPoints(day, ptsBuf)
			oldDist := day.DistTotal

			for i, j := a, b; i < j; i, j = i+1, j-1 {
				ptsBuf[i], ptsBuf[j] = ptsBuf[j], ptsBuf[i]
			}

			feasible, newDist := evalDayFast(sol.Instance, ptsBuf)
			if feasible && newDist < oldDist {
				_, _, newSteps := evalDay(sol.Instance, ptsBuf)
				bestSol.Days[dayIdx].Steps = newSteps
				bestSol.TotalDist += newDist - oldDist
				bestSol.Days[dayIdx].DistTotal = newDist
				improvements++
				sinceLastImprove = 0
			}
		}
	}

	elapsed := time.Since(start)
	fmt.Printf(">> Terminé en %.2fs : %d itérations, %d améliorations.\n", elapsed.Seconds(), iterations, improvements)

	bestSol.EvaluateScore()
	return bestSol
}
