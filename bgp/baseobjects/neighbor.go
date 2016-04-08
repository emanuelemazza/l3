// neighbor.go
package base

import (
	"fmt"
	"l3/bgp/config"
	"l3/bgp/packet"
	"net"
	"utils/logging"
)

type NeighborConf struct {
	logger      *logging.Writer
	Global      *config.GlobalConfig
	Group       *config.PeerGroupConfig
	Neighbor    *config.Neighbor
	BGPId       net.IP
	ASSize      uint8
	AfiSafiMap  map[uint32]bool
	RunningConf config.NeighborConfig
}

func NewNeighborConf(logger *logging.Writer, globalConf *config.GlobalConfig, peerGroup *config.PeerGroupConfig,
	peerConf config.NeighborConfig) *NeighborConf {
	conf := NeighborConf{
		logger:      logger,
		Global:      globalConf,
		Group:       peerGroup,
		AfiSafiMap:  make(map[uint32]bool),
		BGPId:       net.IP{},
		RunningConf: config.NeighborConfig{},
		Neighbor: &config.Neighbor{
			NeighborAddress: peerConf.NeighborAddress,
			Config:          peerConf,
		},
	}

	conf.SetRunningConf(peerGroup, &conf.RunningConf)
	conf.SetNeighborState(&conf.RunningConf)

	if conf.RunningConf.LocalAS == conf.RunningConf.PeerAS {
		conf.Neighbor.State.PeerType = config.PeerTypeInternal
	} else {
		conf.Neighbor.State.PeerType = config.PeerTypeExternal
	}
	if conf.RunningConf.BfdEnable {
		conf.Neighbor.State.BfdNeighborState = "up"
	} else {
		conf.Neighbor.State.BfdNeighborState = "down"
	}

	conf.AfiSafiMap, _ = packet.GetProtocolFromConfig(&conf.Neighbor.AfiSafis)
	return &conf
}

func (n *NeighborConf) SetNeighborState(peerConf *config.NeighborConfig) {
	n.Neighbor.State = config.NeighborState{
		PeerAS:                  peerConf.PeerAS,
		LocalAS:                 peerConf.LocalAS,
		AuthPassword:            peerConf.AuthPassword,
		Description:             peerConf.Description,
		NeighborAddress:         peerConf.NeighborAddress,
		IfIndex:                 peerConf.IfIndex,
		RouteReflectorClusterId: peerConf.RouteReflectorClusterId,
		RouteReflectorClient:    peerConf.RouteReflectorClient,
		MultiHopEnable:          peerConf.MultiHopEnable,
		MultiHopTTL:             peerConf.MultiHopTTL,
		ConnectRetryTime:        peerConf.ConnectRetryTime,
		HoldTime:                peerConf.HoldTime,
		KeepaliveTime:           peerConf.KeepaliveTime,
		PeerGroup:               peerConf.PeerGroup,
		AddPathsRx:              false,
		AddPathsMaxTx:           0,
	}
}

func (n *NeighborConf) UpdateNeighborConf(nConf config.NeighborConfig, bgp *config.Bgp) {
	n.Neighbor.NeighborAddress = nConf.NeighborAddress
	n.Neighbor.Config = nConf
	n.RunningConf = config.NeighborConfig{}
	if nConf.PeerGroup != n.Group.Name {
		if peerGroup, ok := bgp.PeerGroups[nConf.PeerGroup]; ok {
			n.GetNeighConfFromPeerGroup(&peerGroup.Config, &n.RunningConf)
		} else {
			n.logger.Err(fmt.Sprintln("Peer group", nConf.PeerGroup, "not found in BGP config"))
		}
	}
	n.GetConfFromNeighbor(&n.Neighbor.Config, &n.RunningConf)
	n.SetNeighborState(&n.RunningConf)
}

func (n *NeighborConf) UpdatePeerGroup(peerGroup *config.PeerGroupConfig) {
	n.Group = peerGroup
	n.RunningConf = config.NeighborConfig{}
	n.SetRunningConf(peerGroup, &n.RunningConf)
	n.SetNeighborState(&n.RunningConf)
}

func (n *NeighborConf) SetRunningConf(peerGroup *config.PeerGroupConfig, peerConf *config.NeighborConfig) {
	n.GetNeighConfFromGlobal(peerConf)
	n.GetNeighConfFromPeerGroup(peerGroup, peerConf)
	n.GetConfFromNeighbor(&n.Neighbor.Config, peerConf)
}

func (n *NeighborConf) GetNeighConfFromGlobal(peerConf *config.NeighborConfig) {
	peerConf.LocalAS = n.Global.AS
}

func (n *NeighborConf) GetNeighConfFromPeerGroup(groupConf *config.PeerGroupConfig, peerConf *config.NeighborConfig) {
	globalAS := peerConf.LocalAS
	if groupConf != nil {
		peerConf.BaseConfig = groupConf.BaseConfig
	}
	if peerConf.LocalAS == 0 {
		peerConf.LocalAS = globalAS
	}
}

func (n *NeighborConf) GetConfFromNeighbor(inConf *config.NeighborConfig, outConf *config.NeighborConfig) {
	if inConf.PeerAS != 0 {
		outConf.PeerAS = inConf.PeerAS
	}

	if inConf.LocalAS != 0 {
		outConf.LocalAS = inConf.LocalAS
	}

	if inConf.AuthPassword != "" {
		outConf.AuthPassword = inConf.AuthPassword
	}

	if inConf.Description != "" {
		outConf.Description = inConf.Description
	}

	if inConf.RouteReflectorClusterId != 0 {
		outConf.RouteReflectorClusterId = inConf.RouteReflectorClusterId
	}

	if inConf.RouteReflectorClient != false {
		outConf.RouteReflectorClient = inConf.RouteReflectorClient
	}

	if inConf.MultiHopEnable != false {
		outConf.MultiHopEnable = inConf.MultiHopEnable
	}

	if inConf.MultiHopTTL != 0 {
		outConf.MultiHopTTL = inConf.MultiHopTTL
	}

	if inConf.ConnectRetryTime != 0 {
		outConf.ConnectRetryTime = inConf.ConnectRetryTime
	}

	if inConf.HoldTime != 0 {
		outConf.HoldTime = inConf.HoldTime
	}

	if inConf.KeepaliveTime != 0 {
		outConf.KeepaliveTime = inConf.KeepaliveTime
	}

	if inConf.AddPathsRx != false {
		outConf.AddPathsRx = inConf.AddPathsRx
	}

	if inConf.AddPathsMaxTx != 0 {
		outConf.AddPathsMaxTx = inConf.AddPathsMaxTx
	}

	if inConf.BfdEnable != false {
		outConf.BfdEnable = inConf.BfdEnable
	}

	outConf.NeighborAddress = inConf.NeighborAddress
	outConf.IfIndex = inConf.IfIndex
	outConf.PeerGroup = inConf.PeerGroup
}

func (n *NeighborConf) IsInternal() bool {
	return n.RunningConf.PeerAS == n.RunningConf.LocalAS
}

func (n *NeighborConf) IsExternal() bool {
	return n.RunningConf.LocalAS != n.RunningConf.PeerAS
}

func (n *NeighborConf) IsRouteReflectorClient() bool {
	return n.RunningConf.RouteReflectorClient
}

func (n *NeighborConf) FSMStateChange(state uint32) {
	n.logger.Info(fmt.Sprintf("Neighbor %s: FSMStateChange %d", n.Neighbor.NeighborAddress, state))
	n.Neighbor.State.SessionState = uint32(state)
}

func (n *NeighborConf) SetPeerAttrs(bgpId net.IP, asSize uint8, holdTime uint32, keepaliveTime uint32,
	addPathFamily map[packet.AFI]map[packet.SAFI]uint8) {
	n.BGPId = bgpId
	n.ASSize = asSize
	n.Neighbor.State.HoldTime = holdTime
	n.Neighbor.State.KeepaliveTime = keepaliveTime
	for afi, safiMap := range addPathFamily {
		if afi == packet.AfiIP {
			for _, val := range safiMap {
				if (val & packet.BGPCapAddPathRx) != 0 {
					n.logger.Info(fmt.Sprintf("SetPeerAttrs - Neighbor %s set add paths maxtx to %d\n",
						n.Neighbor.NeighborAddress, n.RunningConf.AddPathsMaxTx))
					n.Neighbor.State.AddPathsMaxTx = n.RunningConf.AddPathsMaxTx
				}
				if (val & packet.BGPCapAddPathTx) != 0 {
					n.logger.Info(fmt.Sprintf("SetPeerAttrs - Neighbor %s set add paths rx to %s\n",
						n.Neighbor.NeighborAddress, n.RunningConf.AddPathsRx))
					n.Neighbor.State.AddPathsRx = true
				}
			}
		}
	}
}