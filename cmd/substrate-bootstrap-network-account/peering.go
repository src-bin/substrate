package main

import (
	"log"

	"github.com/src-bin/substrate/veqp"
)

// PeeringConnections ensures we don't duplicate the work of constructing VPC
// peering connections between two quality-region pairs in an environment.
type PeeringConnections map[peeringConnection]bool

func (pcs PeeringConnections) Add(eq0, eq1 veqp.EnvironmentQualityPair, region0, region1 string) {
	pcs[newPeeringConnection(eq0, eq1, region0, region1)] = true
}

func (pcs PeeringConnections) Has(eq0, eq1 veqp.EnvironmentQualityPair, region0, region1 string) bool {
	return pcs[newPeeringConnection(eq0, eq1, region0, region1)]
}

type peeringConnection struct {
	eqs     [2]veqp.EnvironmentQualityPair // must be co-sorted; guaranteed by newPeeringConnection
	regions [2]string                      // must be co-sorted; guaranteed by newPeeringConnection
}

func newPeeringConnection(eq0, eq1 veqp.EnvironmentQualityPair, region0, region1 string) (pc peeringConnection) {
	if eq0.Environment < eq1.Environment {
		pc.eqs = [2]veqp.EnvironmentQualityPair{eq0, eq1}
		pc.regions = [2]string{region0, region1}
	} else if eq0.Environment > eq1.Environment {
		pc.eqs = [2]veqp.EnvironmentQualityPair{eq1, eq0}
		pc.regions = [2]string{region1, region0}
	} else if eq0.Quality < eq1.Quality {
		pc.eqs = [2]veqp.EnvironmentQualityPair{eq0, eq1}
		pc.regions = [2]string{region0, region1}
	} else if eq0.Quality > eq1.Quality {
		pc.eqs = [2]veqp.EnvironmentQualityPair{eq1, eq0}
		pc.regions = [2]string{region1, region0}
	} else if region0 < region1 {
		pc.eqs = [2]veqp.EnvironmentQualityPair{eq0, eq1}
		pc.regions = [2]string{region0, region1}
	} else if region0 > region1 {
		pc.eqs = [2]veqp.EnvironmentQualityPair{eq1, eq0}
		pc.regions = [2]string{region1, region0}
	} else {
		log.Fatalf("can't peer %s %s %s with itself", eq0.Environment, eq0.Quality, region0)
	}
	return
}
