package subnet

import (
	"errors"
	"math"
	"net"

	"github.com/mhuisi/ipv4utils"
)

type Subnet struct {
	AvailabilityZones []AvailabilityZone
	VPC               *net.IPNet
}

type AvailabilityZone struct {
	Public    *net.IPNet
	Private   *net.IPNet
	Protected *net.IPNet
	AZBlock   *net.IPNet
}

func New(ipnet *net.IPNet, azs int) (*Subnet, error) {

	// to split the block into N availability zones evenly,
	// each zone needs to contain 1/n of the available IPs.
	// since each level of the block is split into 2^(level+1) even pieces,
	// if we have 8 AZs, we need to choose the 2nd level -- as 2^(2+1) = 8.

	// this formula only works when AZs are a power of two.
	start := int(math.Log2(float64(azs)))
	if math.Exp2(float64(start)) != float64(azs) {
		return nil, errors.New("AZs must be a power of 2")
	}

	block, err := divideSubnets(ipnet, start+3)
	if err != nil {
		return nil, err
	}

	var subnet Subnet
	subnet.VPC = block[0][0]

	for az := 0; az < azs; az++ {
		subnet.AvailabilityZones = append(subnet.AvailabilityZones, AvailabilityZone{
			AZBlock:   block[start][0+az*1],
			Private:   block[start+1][0+az*2],
			Public:    block[start+2][2+az*4],
			Protected: block[start+3][6+az*8],
		})
	}

	return &subnet, nil
}

func divideSubnets(ipnet *net.IPNet, depth int) ([][]*net.IPNet, error) {
	block := make([][]*net.IPNet, depth+1) // slice of slices of IPNets
	_, _ = ipnet.Mask.Size()               // TODO subnet

	block[0] = []*net.IPNet{ipnet}

	for i := 1; i < depth+1; i++ {
		schan, err := ipv4utils.Subnet(*ipnet, uint(i))
		if err != nil {
			return nil, err
		}
		block[i] = make([]*net.IPNet, 0)
		for subnet := range schan {
			// make a copy of subnet, because memory reuse :(
			var dupSubnet net.IPNet

			dupIp := make(net.IP, len(subnet.IP))
			copy(dupIp, subnet.IP)
			dupSubnet.IP = dupIp

			dupMask := make(net.IPMask, len(subnet.Mask))
			copy(dupMask, subnet.Mask)
			dupSubnet.Mask = dupMask
			block[i] = append(block[i], &dupSubnet)
		}
	}

	return block, nil
}