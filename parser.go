package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// LoadInstance lit un fichier .txt d'instance et retourne une structure Instance remplie.
func LoadInstance(filePath string) (*Instance, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("impossible d'ouvrir le fichier : %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Scan until we find the first non-empty line (Header)
	var headerLine string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			headerLine = line
			break
		}
	}

	if headerLine == "" {
		return nil, fmt.Errorf("fichier vide ou en-tête manquant")
	}

	// Parsing Header: N H D
	parts := strings.Fields(headerLine)
	if len(parts) < 3 {
		return nil, fmt.Errorf("en-tête invalide: %s", headerLine)
	}
	// N, _ := strconv.Atoi(parts[0])
	_, _ = strconv.Atoi(parts[1]) // hCount (unused)
	dCount, _ := strconv.Atoi(parts[2])

	// Parsing Tmax (Line 2)
	var tMax float64
	found := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			tMax, err = strconv.ParseFloat(line, 64)
			if err != nil {
				return nil, fmt.Errorf("Tmax invalide: %v", err)
			}
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("Tmax manquant")
	}

	// Parsing Td (Line 3) - Array of day max distances
	// models.go n'a pas de champ pour les distances par jour, on stocke Tmax dans MaxDist pour le moment.
	found = false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			// On lit juste la ligne pour avancer, on pourra parser si models.go évolue
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("Td manquant")
	}

	// Parsing Points
	var points []*Point
	var hotelIDs []int
	var siteIDs []int

	idCounter := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Format: name x y Si St Oi Ci
		fields := strings.Fields(line)
		if len(fields) < 7 {
			continue // ou erreur
		}

		name := fields[0]
		x, _ := strconv.ParseFloat(fields[1], 64)
		y, _ := strconv.ParseFloat(fields[2], 64)
		score, _ := strconv.ParseFloat(fields[3], 64)
		serviceTime, _ := strconv.ParseFloat(fields[4], 64)
		openTime, _ := strconv.ParseFloat(fields[5], 64)
		closeTime, _ := strconv.ParseFloat(fields[6], 64)

		p := &Point{
			ID:          idCounter,
			X:           x,
			Y:           y,
			Score:       score,
			ServiceTime: serviceTime,
			OpenTime:    openTime,
			CloseTime:   closeTime,
		}

		// Détermination du type
		if strings.HasPrefix(name, "H") {
			p.Type = TypeHotel
			hotelIDs = append(hotelIDs, idCounter)
		} else {
			p.Type = TypeSite
			siteIDs = append(siteIDs, idCounter)
		}

		points = append(points, p)
		idCounter++
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Création de l'instance
	inst := &Instance{
		Name:     filePath, // Ou extraire le nom du fichier
		NbDays:   dCount,
		MaxDist:  tMax,
		Points:   points,
		HotelIDs: hotelIDs,
		SiteIDs:  siteIDs,
	}

	// Initialisation de la matrice de distance
	// Note: NewInstance dans models.go alloue les maps, mais ici on a reconstruit la struct manuellement.
	// On doit allouer la matrice.
	inst.DistMatrix = make([][]float64, len(points))
	for i := range inst.DistMatrix {
		inst.DistMatrix[i] = make([]float64, len(points))
	}
	inst.ComputeDistMatrix()

	// Assign Start/End Hotels
	// line 1 of data -> Start Hotel (Index 0)
	// line 2 of data -> End Hotel (Index 1)
	if len(points) >= 2 {
		inst.StartHotelID = 0
		inst.EndHotelID = 1
	}

	return inst, nil
}
