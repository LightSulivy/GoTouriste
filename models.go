package main

import (
	"math"
)

// Constantes

type NodeType int

const (
	TypeHotel NodeType = iota
	TypeSite
)

// Définition du Problème

type Point struct {
	ID          int
	Type        NodeType
	X, Y        float64
	Score       float64
	OpenTime    float64
	CloseTime   float64
	ServiceTime float64
}

type Instance struct {
	Name string

	NbDays  int
	MaxDist float64

	Points []*Point

	HotelIDs []int
	SiteIDs  []int

	DistMatrix [][]float64

	StartHotelID int
	EndHotelID   int
}

// Définition de la Solution

type Step struct {
	PointID      int
	Arrival      float64
	Wait         float64
	Departure    float64
	DistFromPrev float64
}

type DayTour struct {
	Steps     []Step
	DistTotal float64
	TimeTotal float64
}

type Solution struct {
	Instance *Instance
	Days     []DayTour

	TotalScore float64
	TotalDist  float64
}

// Initialisation de l'instance

func NewInstance(nbPoints int) *Instance {
	inst := &Instance{
		Points:     make([]*Point, nbPoints),
		DistMatrix: make([][]float64, nbPoints),
		HotelIDs:   make([]int, 0),
		SiteIDs:    make([]int, 0),
	}
	for i := range inst.DistMatrix {
		inst.DistMatrix[i] = make([]float64, nbPoints)
	}
	return inst
}

func Distance(p1, p2 *Point) float64 {
	return math.Sqrt((p1.X-p2.X)*(p1.X-p2.X) + (p1.Y-p2.Y)*(p1.Y-p2.Y))
}

func (inst *Instance) ComputeDistMatrix() {
	for i := 0; i < len(inst.Points); i++ {
		inst.DistMatrix[i][i] = 0
		for j := i + 1; j < len(inst.Points); j++ {
			dist := Distance(inst.Points[i], inst.Points[j])
			inst.DistMatrix[i][j] = dist
			inst.DistMatrix[j][i] = dist
		}
	}
}

func (s *Solution) Clone() *Solution {
	newSol := &Solution{
		Instance:   s.Instance,
		TotalScore: s.TotalScore,
		TotalDist:  s.TotalDist,
		Days:       make([]DayTour, len(s.Days)),
	}
	for i, day := range s.Days {
		newDay := DayTour{
			DistTotal: day.DistTotal,
			TimeTotal: day.TimeTotal,
			Steps:     make([]Step, len(day.Steps)),
		}
		copy(newDay.Steps, day.Steps)
		newSol.Days[i] = newDay
	}
	return newSol
}
