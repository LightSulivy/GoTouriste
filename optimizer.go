package main

import (
	"math/rand"
	"runtime"
	"sync"
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

	return &SearchState{sol: sol, visited: visited, unvisited: unvisited}
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

func evalDay(inst *Instance, dayIdx int, dayPoints []int) (bool, float64, []Step) {
	dist := 0.0
	t := 0.0
	maxDist := inst.DayMaxDist(dayIdx)
	steps := make([]Step, len(dayPoints))

	for i, pID := range dayPoints {
		pt := inst.Points[pID]
		steps[i].PointID = pID

		if i > 0 {
			d := inst.DistMatrix[dayPoints[i-1]][pID]
			dist += d
			t += d
			if t > maxDist {
				return false, 0, nil
			}
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
			if t > maxDist {
				return false, 0, nil
			}
		}
		steps[i].Departure = t
	}
	return true, dist, steps
}

func evalDayFast(inst *Instance, dayIdx int, dayPoints []int) (bool, float64) {
	dist := 0.0
	t := 0.0
	maxDist := inst.DayMaxDist(dayIdx)

	for i, pID := range dayPoints {
		pt := inst.Points[pID]
		if i > 0 {
			d := inst.DistMatrix[dayPoints[i-1]][pID]
			dist += d
			t += d
			if t > maxDist {
				return false, 0
			}
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
			if t > maxDist {
				return false, 0
			}
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

// --- Voisinage 1 : Best Insertion ---

func bestInsertion(sol *Solution, state *SearchState) bool {
	inst := sol.Instance
	bestScore := 0.0
	bestDist := 0.0
	bestDayIdx := -1
	bestSiteID := -1
	var bestSteps []Step

	ptsBuf := make([]int, 0, 128)
	newPtsBuf := make([]int, 0, 128)

	for _, u := range state.unvisited {
		uScore := inst.Points[u].Score
		if uScore <= 0 {
			continue
		}
		for dIdx := range sol.Days {
			day := &sol.Days[dIdx]
			ptsBuf = extractDayPoints(day, ptsBuf)

			for pos := 1; pos < len(ptsBuf); pos++ {
				// PRUNING : Évaluation ultra-rapide de la distance
				prevID := ptsBuf[pos-1]
				nextID := ptsBuf[pos] // pos est garanti < len car ptsBuf inclut EndHotel
				deltaDist := inst.DistMatrix[prevID][u] + inst.DistMatrix[u][nextID] - inst.DistMatrix[prevID][nextID]
				if day.DistTotal+deltaDist > inst.DayMaxDist(dIdx) {
					continue
				}

				newPtsBuf = newPtsBuf[:0]
				newPtsBuf = append(newPtsBuf, ptsBuf[:pos]...)
				newPtsBuf = append(newPtsBuf, u)
				newPtsBuf = append(newPtsBuf, ptsBuf[pos:]...)

				feasible, newDist := evalDayFast(inst, dIdx, newPtsBuf)
				if !feasible {
					continue
				}
				if uScore > bestScore || (uScore == bestScore && newDist < bestDist) {
					_, _, steps := evalDay(inst, dIdx, newPtsBuf)
					bestScore = uScore
					bestDist = newDist
					bestDayIdx = dIdx
					bestSiteID = u
					bestSteps = steps
				}
			}
		}
	}

	if bestDayIdx < 0 {
		return false
	}
	sol.Days[bestDayIdx].Steps = bestSteps
	sol.Days[bestDayIdx].DistTotal = bestDist
	sol.TotalScore += bestScore
	state.markVisited(bestSiteID)
	return true
}

// --- Voisinage 2 : 2-opt exhaustif ---

func apply2OptAllDays(sol *Solution) bool {
	inst := sol.Instance
	improved := false
	ptsBuf := make([]int, 0, 128)

	for dIdx := range sol.Days {
		day := &sol.Days[dIdx]
		if len(day.Steps) <= 3 {
			continue
		}

		dayImproved := true
		for dayImproved {
			dayImproved = false
			ptsBuf = extractDayPoints(day, ptsBuf)
			n := len(ptsBuf)

			for i := 1; i < n-2; i++ {
				for j := i + 1; j < n-1; j++ {
					for a, b := i, j; a < b; a, b = a+1, b-1 {
						ptsBuf[a], ptsBuf[b] = ptsBuf[b], ptsBuf[a]
					}
					feasible, newDist := evalDayFast(inst, dIdx, ptsBuf)
					if feasible && newDist < day.DistTotal {
						_, _, newSteps := evalDay(inst, dIdx, ptsBuf)
						day.Steps = newSteps
						day.DistTotal = newDist
						dayImproved = true
						improved = true
						break
					}
					for a, b := i, j; a < b; a, b = a+1, b-1 {
						ptsBuf[a], ptsBuf[b] = ptsBuf[b], ptsBuf[a]
					}
				}
				if dayImproved {
					break
				}
			}
		}
	}
	return improved
}

// --- Voisinage 3 : Best Swap ---

func bestSwap(sol *Solution, state *SearchState) bool {
	inst := sol.Instance
	bestDelta := 0.0
	bestDist := 0.0
	bestDayIdx := -1
	bestNewID := -1
	bestOldID := -1
	var bestSteps []Step

	ptsBuf := make([]int, 0, 128)

	for dIdx := range sol.Days {
		day := &sol.Days[dIdx]
		if len(day.Steps) <= 2 {
			continue
		}
		ptsBuf = extractDayPoints(day, ptsBuf)

		for pos := 1; pos < len(ptsBuf)-1; pos++ {
			oldID := ptsBuf[pos]
			if inst.Points[oldID].Type != TypeSite {
				continue
			}
			prevID := ptsBuf[pos-1]
			nextID := ptsBuf[pos+1]

			for _, u := range state.unvisited {
				delta := inst.Points[u].Score - inst.Points[oldID].Score
				if delta < bestDelta {
					continue
				}

				// PRUNING : Évaluation ultra-rapide de la distance
				deltaDist := (inst.DistMatrix[prevID][u] + inst.DistMatrix[u][nextID]) - (inst.DistMatrix[prevID][oldID] + inst.DistMatrix[oldID][nextID])
				if day.DistTotal+deltaDist > inst.DayMaxDist(dIdx) {
					continue
				}

				ptsBuf[pos] = u
				feasible, newDist := evalDayFast(inst, dIdx, ptsBuf)
				ptsBuf[pos] = oldID
				if !feasible {
					continue
				}
				if delta > bestDelta || (delta == bestDelta && newDist < bestDist) {
					ptsBuf[pos] = u
					_, _, steps := evalDay(inst, dIdx, ptsBuf)
					ptsBuf[pos] = oldID
					bestDelta = delta
					bestDist = newDist
					bestDayIdx = dIdx
					bestNewID = u
					bestOldID = oldID
					bestSteps = steps
				}
			}
		}
	}

	if bestDayIdx < 0 || bestDelta <= 0 {
		return false
	}
	sol.Days[bestDayIdx].Steps = bestSteps
	sol.Days[bestDayIdx].DistTotal = bestDist
	sol.TotalScore += bestDelta
	state.markVisited(bestNewID)
	state.markUnvisited(bestOldID)
	return true
}

// --- Voisinage 4 : Relocate (intra + inter-jour) ---

func applyRelocate(sol *Solution, state *SearchState, rng *rand.Rand, maxIter int) bool {
	inst := sol.Instance
	improved := false
	ptsBuf := make([]int, 0, 128)
	newPtsBuf := make([]int, 0, 128)
	trialBuf := make([]int, 0, 128)
	nbDays := len(sol.Days)
	if nbDays == 0 {
		return false
	}

	for iter := 0; iter < maxIter; iter++ {
		dayIdx := rng.Intn(nbDays)
		day := &sol.Days[dayIdx]
		nbSteps := len(day.Steps)
		if nbSteps <= 2 {
			continue
		}

		moveType := rng.Intn(3)

		if moveType == 0 && nbSteps > 3 {
			pos1 := 1 + rng.Intn(nbSteps-2)
			pos2 := 1 + rng.Intn(nbSteps-2)
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
			feasible, newDist := evalDayFast(inst, dayIdx, newPtsBuf)
			if feasible && newDist < oldDist {
				_, _, newSteps := evalDay(inst, dayIdx, newPtsBuf)
				sol.Days[dayIdx].Steps = newSteps
				sol.Days[dayIdx].DistTotal = newDist
				improved = true
			}

		} else if moveType == 1 && nbDays > 1 && nbSteps > 2 {
			pos := 1 + rng.Intn(nbSteps-2)
			siteID := day.Steps[pos].PointID
			if inst.Points[siteID].Type != TypeSite {
				continue
			}
			destDay := rng.Intn(nbDays)
			if destDay == dayIdx {
				continue
			}
			ptsBuf = extractDayPoints(day, ptsBuf)
			srcPts := make([]int, 0, len(ptsBuf)-1)
			srcPts = append(srcPts, ptsBuf[:pos]...)
			srcPts = append(srcPts, ptsBuf[pos+1:]...)
			srcOk, srcDist := evalDayFast(inst, dayIdx, srcPts)
			if !srcOk {
				continue
			}
			destDayPtr := &sol.Days[destDay]
			newPtsBuf = extractDayPoints(destDayPtr, newPtsBuf)
			destLen := len(newPtsBuf)
			if destLen < 2 {
				continue
			}
			bestDestDist := -1.0
			bestDestPos := -1
			for p := 1; p < destLen; p++ {
				// PRUNING : Évaluation ultra-rapide de la distance
				prevID := newPtsBuf[p-1]
				nextID := newPtsBuf[p]
				deltaDist := inst.DistMatrix[prevID][siteID] + inst.DistMatrix[siteID][nextID] - inst.DistMatrix[prevID][nextID]
				if destDayPtr.DistTotal+deltaDist > inst.DayMaxDist(destDay) {
					continue
				}

				trialBuf = trialBuf[:0]
				trialBuf = append(trialBuf, newPtsBuf[:p]...)
				trialBuf = append(trialBuf, siteID)
				trialBuf = append(trialBuf, newPtsBuf[p:]...)
				ok, d := evalDayFast(inst, destDay, trialBuf)
				if ok && (bestDestPos == -1 || d < bestDestDist) {
					bestDestDist = d
					bestDestPos = p
				}
			}
			if bestDestPos == -1 {
				continue
			}
			_, _, srcSteps := evalDay(inst, dayIdx, srcPts)
			sol.Days[dayIdx].Steps = srcSteps
			sol.Days[dayIdx].DistTotal = srcDist
			destPts := make([]int, 0, destLen+1)
			destPts = append(destPts, newPtsBuf[:bestDestPos]...)
			destPts = append(destPts, siteID)
			destPts = append(destPts, newPtsBuf[bestDestPos:]...)
			_, destDist, destSteps := evalDay(inst, destDay, destPts)
			sol.Days[destDay].Steps = destSteps
			sol.Days[destDay].DistTotal = destDist
			improved = true

		} else if len(state.unvisited) > 0 && nbSteps >= 2 {
			uIdx := rng.Intn(len(state.unvisited))
			u := state.unvisited[uIdx]
			pos := 1 + rng.Intn(nbSteps-1)
			ptsBuf = extractDayPoints(day, ptsBuf)

			// PRUNING : Évaluation ultra-rapide de la distance
			prevID := ptsBuf[pos-1]
			nextID := ptsBuf[pos]
			deltaDist := inst.DistMatrix[prevID][u] + inst.DistMatrix[u][nextID] - inst.DistMatrix[prevID][nextID]
			if day.DistTotal+deltaDist > inst.DayMaxDist(dayIdx) {
				continue
			}

			newPtsBuf = newPtsBuf[:0]
			newPtsBuf = append(newPtsBuf, ptsBuf[:pos]...)
			newPtsBuf = append(newPtsBuf, u)
			newPtsBuf = append(newPtsBuf, ptsBuf[pos:]...)
			feasible, newDist := evalDayFast(inst, dayIdx, newPtsBuf)
			if feasible {
				_, _, newSteps := evalDay(inst, dayIdx, newPtsBuf)
				sol.Days[dayIdx].Steps = newSteps
				sol.Days[dayIdx].DistTotal = newDist
				sol.TotalScore += inst.Points[u].Score
				state.markVisited(u)
				improved = true
			}
		}
	}
	return improved
}

// --- Voisinage 5 : Hotel Swap ---

func tryHotelSwap(sol *Solution, rng *rand.Rand) bool {
	inst := sol.Instance
	if inst.NbDays <= 1 {
		return false
	}

	improved := false

	// Pour chaque jonction entre jours d et d+1
	for d := 0; d < len(sol.Days)-1; d++ {
		day := &sol.Days[d]
		nextDay := &sol.Days[d+1]

		if len(day.Steps) < 2 || len(nextDay.Steps) < 2 {
			continue
		}

		// L'hôtel actuel est le dernier step du jour d = premier step du jour d+1
		currentHotelID := day.Steps[len(day.Steps)-1].PointID

		bestScore := sol.TotalScore
		bestHotel := -1

		for _, hID := range inst.HotelIDs {
			if hID == currentHotelID {
				continue
			}
			// Le premier jour doit terminer à hID, le jour suivant doit commencer à hID
			// On teste si les journées sont faisables avec ce nouvel hôtel

			// Reconstruire les points du jour d avec le nouvel hôtel de fin
			ptsDayD := make([]int, len(day.Steps))
			for i, s := range day.Steps {
				ptsDayD[i] = s.PointID
			}
			ptsDayD[len(ptsDayD)-1] = hID

			feasD, distD := evalDayFast(inst, d, ptsDayD)
			if !feasD {
				continue
			}

			// Reconstruire les points du jour d+1 avec le nouvel hôtel de départ
			ptsNextDay := make([]int, len(nextDay.Steps))
			for i, s := range nextDay.Steps {
				ptsNextDay[i] = s.PointID
			}
			ptsNextDay[0] = hID

			feasN, distN := evalDayFast(inst, d+1, ptsNextDay)
			if !feasN {
				continue
			}

			// Si les deux jours sont faisables et la distance totale diminue
			totalNewDist := distD + distN
			totalOldDist := day.DistTotal + nextDay.DistTotal
			if totalNewDist < totalOldDist {
				bestHotel = hID
				_ = bestScore
			}
		}

		if bestHotel >= 0 {
			// Appliquer le changement d'hôtel
			ptsDayD := make([]int, len(day.Steps))
			for i, s := range day.Steps {
				ptsDayD[i] = s.PointID
			}
			ptsDayD[len(ptsDayD)-1] = bestHotel

			_, distD, stepsD := evalDay(inst, d, ptsDayD)
			day.Steps = stepsD
			day.DistTotal = distD

			ptsNextDay := make([]int, len(nextDay.Steps))
			for i, s := range nextDay.Steps {
				ptsNextDay[i] = s.PointID
			}
			ptsNextDay[0] = bestHotel

			_, distN, stepsN := evalDay(inst, d+1, ptsNextDay)
			nextDay.Steps = stepsN
			nextDay.DistTotal = distN

			improved = true
		}
	}

	sol.EvaluateScore()
	return improved
}

// --- VND : Variable Neighbourhood Descent ---

func applyVND(sol *Solution, rng *rand.Rand, deadline time.Time) {
	k := 0
	for k < 5 {
		if time.Now().After(deadline) {
			break
		}
		oldScore := sol.TotalScore

		switch k {
		case 0:
			// Insertion : Tente d'ajouter un site non visité pour augmenter le score
			state := newSearchState(sol)
			for bestInsertion(sol, state) {
				if time.Now().After(deadline) {
					break
				}
			}
			sol.EvaluateScore()
		case 1:
			// 2-Opt : Décroise les chemins dans une même journée pour gagner de la distance utile
			apply2OptAllDays(sol)
			sol.EvaluateScore()
		case 2:
			// Échange (Swap) : Remplace un site prévu par un meilleur non visité
			state := newSearchState(sol)
			for bestSwap(sol, state) {
				if time.Now().After(deadline) {
					break
				}
			}
			sol.EvaluateScore()
		case 3:
			// Déplacement (Relocate) : Réorganise les sites (même jour/inter-jour) pour boucher les trous
			state := newSearchState(sol)
			applyRelocate(sol, state, rng, 2000)
			sol.EvaluateScore()
		case 4:
			// Hôtel (Hotel Swap) : Raccourcit les trajets fin/début de journée en changeant l'hôtel
			tryHotelSwap(sol, rng)
		}

		if sol.TotalScore > oldScore {
			k = 0
		} else {
			k++
		}
	}
}

// --- Shaking : Destruction aléatoire d'une solution ---

func applyShake(sol *Solution, force int, rng *rand.Rand) {
	inst := sol.Instance

	switch {
	case force <= 1:
		// Force 1 : Retirer 2-3 sites aléatoires
		nbRemove := 2 + rng.Intn(2)
		removeSitesRandom(sol, inst, nbRemove, rng)

	case force == 2:
		// Force 2 : Retirer des sites + changer un hôtel intermédiaire
		totalSites := 0
		for _, day := range sol.Days {
			for _, s := range day.Steps {
				if inst.Points[s.PointID].Type == TypeSite {
					totalSites++
				}
			}
		}
		nbRemove := totalSites / 8
		if nbRemove < 3 {
			nbRemove = 3
		}
		removeSitesRandom(sol, inst, nbRemove, rng)

		// Changer un hôtel intermédiaire au hasard
		if inst.NbDays > 1 && len(inst.HotelIDs) > 1 {
			junctionIdx := rng.Intn(inst.NbDays - 1) // jonction entre jour j et j+1
			day := &sol.Days[junctionIdx]
			nextDay := &sol.Days[junctionIdx+1]
			if len(day.Steps) >= 2 && len(nextDay.Steps) >= 2 {
				newHotel := inst.HotelIDs[rng.Intn(len(inst.HotelIDs))]
				day.Steps[len(day.Steps)-1].PointID = newHotel
				nextDay.Steps[0].PointID = newHotel
				// Vider les sites des 2 jours affectés pour laisser le VND reconstruire
				clearDay(sol, inst, junctionIdx)
				clearDay(sol, inst, junctionIdx+1)
			}
		}

	case force == 3:
		// Force 3 : Vider complètement un jour aléatoire
		if len(sol.Days) > 1 {
			dayIdx := rng.Intn(len(sol.Days))
			day := &sol.Days[dayIdx]
			newSteps := make([]Step, 0, 2)
			for _, s := range day.Steps {
				if inst.Points[s.PointID].Type == TypeHotel {
					newSteps = append(newSteps, s)
				}
			}
			day.Steps = newSteps
			day.DistTotal = 0
			if len(newSteps) >= 2 {
				d := inst.DistMatrix[newSteps[0].PointID][newSteps[len(newSteps)-1].PointID]
				day.DistTotal = d
			}
		}

	default:
		// Force 4+ : NUCLEAR - Vider TOUS les jours (garder uniquement les hôtels)
		// Le VND reconstruira tout via Best Insertion
		for dIdx := range sol.Days {
			clearDay(sol, inst, dIdx)
		}
	}

	sol.EvaluateScore()
}

func removeSitesRandom(sol *Solution, inst *Instance, count int, rng *rand.Rand) {
	// Collecte de tous les sites visitables
	type sitePos struct {
		dayIdx  int
		stepIdx int
	}
	var positions []sitePos
	for dIdx, day := range sol.Days {
		for sIdx, s := range day.Steps {
			if inst.Points[s.PointID].Type == TypeSite {
				positions = append(positions, sitePos{dIdx, sIdx})
			}
		}
	}
	if len(positions) == 0 {
		return
	}

	// Mélanger et retirer les premiers 'count'
	rng.Shuffle(len(positions), func(i, j int) {
		positions[i], positions[j] = positions[j], positions[i]
	})
	if count > len(positions) {
		count = len(positions)
	}

	// Marquer les steps à supprimer (par jour, en ordre décroissant d'index)
	removals := make(map[int][]int)
	for i := 0; i < count; i++ {
		p := positions[i]
		removals[p.dayIdx] = append(removals[p.dayIdx], p.stepIdx)
	}

	for dIdx, indices := range removals {
		// Trier décroissant pour supprimer sans décaler
		for i := 0; i < len(indices); i++ {
			for j := i + 1; j < len(indices); j++ {
				if indices[j] > indices[i] {
					indices[i], indices[j] = indices[j], indices[i]
				}
			}
		}
		day := &sol.Days[dIdx]
		for _, sIdx := range indices {
			day.Steps = append(day.Steps[:sIdx], day.Steps[sIdx+1:]...)
		}
		// Recalculer la distance du jour
		dist := 0.0
		for i := 1; i < len(day.Steps); i++ {
			dist += inst.DistMatrix[day.Steps[i-1].PointID][day.Steps[i].PointID]
		}
		day.DistTotal = dist
	}
}

func clearDay(sol *Solution, inst *Instance, dayIdx int) {
	day := &sol.Days[dayIdx]
	newSteps := make([]Step, 0, 2)
	for _, s := range day.Steps {
		if inst.Points[s.PointID].Type == TypeHotel {
			newSteps = append(newSteps, s)
		}
	}
	day.Steps = newSteps
	day.DistTotal = 0
	if len(newSteps) >= 2 {
		day.DistTotal = inst.DistMatrix[newSteps[0].PointID][newSteps[len(newSteps)-1].PointID]
	}
}

// --- GRASP + VNS hybride parallèle ---

func LocalSearch(sol *Solution, maxDuration time.Duration, targetScore float64) (*Solution, time.Duration) {
	start := time.Now()
	globalDeadline := start.Add(maxDuration)

	bestSol := sol.Clone()
	bestSol.EvaluateScore()
	rng0 := rand.New(rand.NewSource(42))
	applyVND(bestSol, rng0, globalDeadline)
	bestFoundAt := time.Since(start)

	// Arrêt si optimal déjà trouvé
	if targetScore > 0 && bestSol.TotalScore >= targetScore {
		return bestSol, bestFoundAt
	}

	inst := sol.Instance
	remaining := maxDuration - time.Since(start)

	if remaining < 500*time.Millisecond {
		return bestSol, bestFoundAt
	}

	type workerResult struct {
		sol     *Solution
		foundAt time.Duration
	}

	nbWorkers := runtime.NumCPU()
	if nbWorkers < 4 {
		nbWorkers = 4
	}
	nbGRASP := nbWorkers / 2 // Moitié exploration
	results := make([]workerResult, nbWorkers)
	var wg sync.WaitGroup
	done := make(chan struct{})

	for w := 0; w < nbWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			seed := time.Now().UnixNano() + int64(workerID*1337)
			rng := rand.New(rand.NewSource(seed))
			localBest := bestSol.Clone()
			localBestFoundAt := bestFoundAt
			deadline := time.Now().Add(remaining)

			if workerID < nbGRASP {
				// --- WORKERS GRASP (Exploration large) ---
				rclSize := 2 + (workerID % 5) + 1
				ratioMode := workerID % 4 // 0 = Score/Coût, 1 = Urgence, 2 = Score, 3 = Proximité

				for time.Now().Before(deadline) {
					select {
					case <-done:
						results[workerID] = workerResult{sol: localBest, foundAt: localBestFoundAt}
						return
					default:
					}

					candidate := SolveGreedyRandomized(inst, rclSize, rng, ratioMode)
					candidate.EvaluateScore()
					applyVND(candidate, rng, deadline)

					valid, _ := EvaluateSolution(candidate)
					if !valid {
						continue
					}

					if candidate.TotalScore > localBest.TotalScore ||
						(candidate.TotalScore == localBest.TotalScore && candidate.TotalDist < localBest.TotalDist) {
						localBest = candidate
						localBestFoundAt = time.Since(start)
					}

					if targetScore > 0 && localBest.TotalScore >= targetScore {
						results[workerID] = workerResult{sol: localBest, foundAt: localBestFoundAt}
						select {
						case <-done:
						default:
							close(done)
						}
						return
					}
				}

			} else {
				// --- WORKERS VNS / ILS (Exploitation profonde) ---
				shakeForce := 1
				maxForce := 4
				noImproveCount := 0

				for time.Now().Before(deadline) {
					select {
					case <-done:
						results[workerID] = workerResult{sol: localBest, foundAt: localBestFoundAt}
						return
					default:
					}

					// Shaking : on secoue à partir du meilleur record
					candidate := localBest.Clone()
					applyShake(candidate, shakeForce, rng)

					// VND : on répare et on optimise
					applyVND(candidate, rng, deadline)

					valid, _ := EvaluateSolution(candidate)
					if !valid {
						noImproveCount++
						if noImproveCount > 5 {
							shakeForce++
							if shakeForce > maxForce {
								shakeForce = 1
							}
							noImproveCount = 0
						}
						continue
					}

					if candidate.TotalScore > localBest.TotalScore ||
						(candidate.TotalScore == localBest.TotalScore && candidate.TotalDist < localBest.TotalDist) {
						localBest = candidate
						localBestFoundAt = time.Since(start)
						shakeForce = 1 // Amélioration -> on repart doucement
						noImproveCount = 0
					} else {
						noImproveCount++
						if noImproveCount > 5 {
							shakeForce++
							if shakeForce > maxForce {
								shakeForce = 1
							}
							noImproveCount = 0
						}
					}

					if targetScore > 0 && localBest.TotalScore >= targetScore {
						results[workerID] = workerResult{sol: localBest, foundAt: localBestFoundAt}
						select {
						case <-done:
						default:
							close(done)
						}
						return
					}
				}
			}

			results[workerID] = workerResult{sol: localBest, foundAt: localBestFoundAt}
		}(w)
	}

	wg.Wait()

	for _, r := range results {
		if r.sol == nil {
			continue
		}
		valid, _ := EvaluateSolution(r.sol)
		if !valid {
			continue
		}
		if r.sol.TotalScore > bestSol.TotalScore ||
			(r.sol.TotalScore == bestSol.TotalScore && r.sol.TotalDist < bestSol.TotalDist) {
			bestSol = r.sol
			bestFoundAt = r.foundAt
		}
	}

	bestSol.EvaluateScore()
	return bestSol, bestFoundAt
}
