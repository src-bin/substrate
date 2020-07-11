package main

import "log"

type peeringConnection struct {
	environment        string
	qualities, regions [2]string // must be co-sorted; guaranteed by newPeeringConnection
}

func newPeeringConnection(environment, quality0, region0, quality1, region1 string) (pc peeringConnection) {
	pc.environment = environment
	if quality0 < quality1 {
		pc.qualities = [2]string{quality0, quality1}
		pc.regions = [2]string{region0, region1}
	} else if quality0 > quality1 {
		pc.qualities = [2]string{quality1, quality0}
		pc.regions = [2]string{region1, region0}
	} else if region0 < region1 {
		pc.qualities = [2]string{quality0, quality1}
		pc.regions = [2]string{region0, region1}
	} else if region0 > region1 {
		pc.qualities = [2]string{quality1, quality0}
		pc.regions = [2]string{region1, region0}
	} else {
		log.Println(environment, quality0, region0, quality1, region1)
		log.Fatalf("newPeeringConnection can't peer %s %s with itself", quality0, region0)
	}
	return
}

// PeeringConnections ensures we don't duplicate the work of constructing VPC
// peering connections between two quality-region pairs in an environment.
type PeeringConnections map[peeringConnection]bool

func (pcs PeeringConnections) Add(environment, quality0, region0, quality1, region1 string) {
	pcs[newPeeringConnection(environment, quality0, region0, quality1, region1)] = true
}

func (pcs PeeringConnections) Has(environment, quality0, region0, quality1, region1 string) bool {
	return pcs[newPeeringConnection(environment, quality0, region0, quality1, region1)]
}
