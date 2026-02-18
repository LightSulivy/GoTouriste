package main

// EvaluateScore calcule la somme des scores des sites visités et la distance totale
func (s *Solution) EvaluateScore() {
	var totalScore float64
	var totalDist float64
	for _, day := range s.Days {
		// Recalculer la distance du jour si besoin, ou utiliser ce qui est stocké
		// Si le solver a déjà calculé DistTotal, on l'utilise.
		// Mais pour être sûr :
		distDay := 0.0
		for _, step := range day.Steps {
			distDay += step.DistFromPrev
			if s.Instance.Points[step.PointID].Type == TypeSite {
				totalScore += s.Instance.Points[step.PointID].Score
			}
		}
		totalDist += distDay
	}
	s.TotalScore = totalScore
	s.TotalDist = totalDist
}
