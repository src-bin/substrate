package networks

import (
	"log"

	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/veqp"
)

// PeeringConnections ensures we don't duplicate the work of constructing VPC
// peering connections between two quality-region pairs in an environment.
type PeeringConnections map[peeringConnection]bool

func EnumeratePeeringConnections() (PeeringConnections, error) {
	veqpDoc, err := veqp.ReadDocument()
	if err != nil {
		return nil, err
	}
	pc := PeeringConnections{}
	for _, eq0 := range veqpDoc.ValidEnvironmentQualityPairs {
		for _, region0 := range regions.Selected() {
			for _, eq1 := range veqpDoc.ValidEnvironmentQualityPairs {
				for _, region1 := range regions.Selected() {

					// Don't peer with oneself.
					if eq0 == eq1 && region0 == region1 {
						continue
					}

					// Peer admin networks with everything but otherwise
					// only peer networks with matching environments.
					if eq0.Environment != "admin" && eq0.Environment != eq1.Environment {
						continue
					}

					pc.Add(eq0, eq1, region0, region1)
				}
			}
		}
	}
	// TODO find a way to keep this sorted in the sensible order in which it's constructed
	return pc, nil
}

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

// Ends returns the environment, quality, and region of both ends of the peering connection.
func (pc peeringConnection) Ends() (eq0, eq1 veqp.EnvironmentQualityPair, region0, region1 string) {
	return pc.eqs[0], pc.eqs[1], pc.regions[0], pc.regions[1]
}
