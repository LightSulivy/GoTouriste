package main

import (
	"fmt"
)

func EvaluateSolution(sol *Solution) (bool, error) {
	inst := sol.Instance
	visitedSites := make(map[int]bool)
	totalScore := 0.0
	totalDistGlobal := 0.0

	for dayIdx, day := range sol.Days {
		if len(day.Steps) == 0 {
			continue
		}

		currentDayDist := 0.0
		currentDayTime := 0.0
		firstStep := &sol.Days[dayIdx].Steps[0]
		firstPt := inst.Points[firstStep.PointID]

		if firstPt.Type != TypeHotel {
			return false, fmt.Errorf("Jour %d : Le point de départ (ID %d) n'est pas un hôtel.", dayIdx+1, firstPt.ID)
		}

		if dayIdw == 0 && firstPt.ID != inst.StartHotelID {
			return false, fmt.Errorf("Jour 1 : Départ invalide. Attendu ID %d, reçu ID %d.", inst.StartHotelID, firstPt.ID)
		}

		firstStep.Arrival = 0
		firstStep.Wait = 0
		firstStep.Departure = 0
		firstStep.DistFromPrev = 0

		for i := 1; i < len(day.Steps); i++ {
			prevStep := &sol.Days[dayIdx].Steps[i-1]
			currStep := &sol.Days[dayIdx].Steps[i]
			currPt := inst.Points[currStep.PointID]
			prevPT := inst.Points[prevStep.PointID]

			dist := inst.DistMatrix[prevPt.ID][currPt.ID]
			currentDayDist += dist
			currentDayTime += dist

			currStep.DistFromPrev = dist
			currStep.Arrival = currentDayTime

			if currPt.Type == TypeSite {
				if visitedSites[currPt.ID] {
					return false, fmt.Errorf("Jour %d : Le site %d a déjà été visité", dayIdx+1, currPt.ID)
				}

				wait := 0.0
				if currStep.Arrival < currPt.OpenTime {
					wait = currPt.OpenTime - currStep.Arrival
				}
				currStep.Wait = wait
				startVisit := currStep.Arrival + wait
				
				if startVisit > currPt.CloseTime {
					return false, fmt.Errorf("Jour %d : Arrivée tardive au site %d (Arrivée : %.2f, fermeture : %.2f).", dayIdx+1, currPt.ID, startVisit, currPt.CloseTime)
				}

				visitedSites[currPt.ID] = true
				totalScore += currPt.Score

				currentDayTime = startVisit + currPt.ServiceTime
				currStep.Departure = currentDayTime
			}
			else {
				currStep.Wait = 0
				currStep.Departure = currentDayTime
			}
		}
		lastStep := &sol.Days[dayIdx].Steps[len(day.Steps)-1]
		lastPt := inst.Points[lastStep.PointID]

		if lastPt.Type != TypeHotel {
			return false, fmt.Errorf("Jour %d : Le point d'arrivée (ID %d) n'est pas un hôtel.", dayIdx+1, lastPt.ID)
		}

		if dayIdx == inst.NbDays-1 && lastPt.ID != inst.EndHotelID {
			return false, fmt.Errorf("Dernier jour : Arrivée invalide. Attendu ID %d, reçu ID %d.", inst.EndHotelID, lastPt.ID)
		}

		budgetMax := 0.0

		if dayIdx < len(inst.MaxDist) {
			budgetMax = inst.MaxDist[dayIdx]
		}
		else if len(inst.MaxDist) > 0 {
			budgetMax = inst.MaxDist[0]
		}

		if currentDayDist > budgetMax {
			return false, fmt.Errorf("Jour %d : Budget distance dépassé (%.2f / %.2f).", dayIdx+1, currentDayDist, budgetMax)
		}

		sol.Days[dayIdx].DistTotal = currentDayDist
		sol.Days[dayIdx].TimeTotal = currentDayTime
		totalDistGlobal += currentDayDist
	}
	sol.TotalScore = totalScore
	sol.TotalDist = totalDistGlobal
	return true, nil
}