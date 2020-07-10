package main

import "sort"

type PeeringConnection struct {
	Environment        string
	Qualities, Regions [2]string // must be sorted; guaranteed by newPeeringConnection
}

func newPeeringConnection(environment, quality0, region0, quality1, region1 string) PeeringConnection {
	qualities := [2]string{quality0, quality1}
	sort.Strings(qualities[:])
	regions := [2]string{region0, region1}
	sort.Strings(regions[:])
	return PeeringConnection{
		Environment: environment,
		Qualities:   qualities,
		Regions:     regions,
	}
}

// PeeringConnections ensures we don't duplicate the work of constructing VPC
// peering connections between two quality-region pairs in an environment.
type PeeringConnections map[PeeringConnection]bool

func (pcs PeeringConnections) Add(environment, quality0, region0, quality1, region1 string) {
	pcs[newPeeringConnection(environment, quality0, region0, quality1, region1)] = true
}

func (pcs PeeringConnections) Has(environment, quality0, region0, quality1, region1 string) bool {
	return pcs[newPeeringConnection(environment, quality0, region0, quality1, region1)]
}
