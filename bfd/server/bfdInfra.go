package server

import (
	"asicd/asicdConstDefs"
	"asicdServices"
	"fmt"
	"net"
)

type PortProperty struct {
	Name     string
	VlanName string
	VlanId   uint16
	IpAddr   net.IP
}

type VlanProperty struct {
	Name       string
	UntagPorts []int32
	IpAddr     net.IP
}

type IPIntfProperty struct {
	IfName  string
	IpAddr  net.IP
	MacAddr net.HardwareAddr
	NetMask []byte
}

type IPv4IntfNotifyMsg struct {
	IpAddr string
	IfId   int32
}

func (server *BFDServer) updateIpInVlanPropertyMap(msg IPv4IntfNotifyMsg, msgType uint8) {
	if msgType == asicdConstDefs.NOTIFY_IPV4INTF_CREATE { // Create IP
		ent := server.vlanPropertyMap[msg.IfId]
		ip, _, _ := net.ParseCIDR(msg.IpAddr)
		ent.IpAddr = ip
		server.vlanPropertyMap[msg.IfId] = ent
	} else { // Delete IP
		ent := server.vlanPropertyMap[msg.IfId]
		ent.IpAddr = nil
		server.vlanPropertyMap[msg.IfId] = ent
	}
}

func (server *BFDServer) updateIpInPortPropertyMap(msg IPv4IntfNotifyMsg, msgType uint8) {
	if msgType == asicdConstDefs.NOTIFY_IPV4INTF_CREATE { // Create IP
		ent := server.portPropertyMap[int32(msg.IfId)]
		ip, _, _ := net.ParseCIDR(msg.IpAddr)
		ent.IpAddr = ip
		server.portPropertyMap[int32(msg.IfId)] = ent
	} else { // Delete IP
		ent := server.portPropertyMap[int32(msg.IfId)]
		ent.IpAddr = nil
		server.portPropertyMap[int32(msg.IfId)] = ent
	}
}

func (server *BFDServer) updateVlanPropertyMap(vlanNotifyMsg asicdConstDefs.VlanNotifyMsg, msgType uint8) {
	if msgType == asicdConstDefs.NOTIFY_VLAN_CREATE { // Create Vlan
		ent := server.vlanPropertyMap[int32(vlanNotifyMsg.VlanId)]
		ent.Name = vlanNotifyMsg.VlanName
		ent.UntagPorts = vlanNotifyMsg.UntagPorts
		server.vlanPropertyMap[int32(vlanNotifyMsg.VlanId)] = ent
	} else { // Delete Vlan
		delete(server.vlanPropertyMap, int32(vlanNotifyMsg.VlanId))
	}
}

func (server *BFDServer) updatePortPropertyMap(vlanNotifyMsg asicdConstDefs.VlanNotifyMsg, msgType uint8) {
	if msgType == asicdConstDefs.NOTIFY_VLAN_CREATE { // Create Vlan
		for _, portNum := range vlanNotifyMsg.UntagPorts {
			ent := server.portPropertyMap[portNum]
			ent.VlanId = vlanNotifyMsg.VlanId
			ent.VlanName = vlanNotifyMsg.VlanName
			server.portPropertyMap[portNum] = ent
		}
	} else { // Delete Vlan
		for _, portNum := range vlanNotifyMsg.UntagPorts {
			ent := server.portPropertyMap[portNum]
			ent.VlanId = 0
			ent.VlanName = ""
			server.portPropertyMap[portNum] = ent
		}
	}
}

func (server *BFDServer) BuildPortPropertyMap() {
	currMarker := asicdServices.Int(asicdConstDefs.MIN_SYS_PORTS)
	if server.asicdClient.IsConnected {
		server.logger.Info("Calling asicd for port property")
		count := 10
		for {
			server.logger.Info(fmt.Sprintln("Calling bulkget port ", currMarker, count))
			bulkInfo, _ := server.asicdClient.ClientHdl.GetBulkPortState(asicdServices.Int(currMarker), asicdServices.Int(count))
			if bulkInfo == nil {
				server.logger.Info("Bulkget port got nothing")
				return
			}
			objCount := int(bulkInfo.Count)
			more := bool(bulkInfo.More)
			server.logger.Info(fmt.Sprintln("Bulkget port got ", objCount, more))
			currMarker = asicdServices.Int(bulkInfo.EndIdx)
			for i := 0; i < objCount; i++ {
				portNum := bulkInfo.PortStateList[i].PortNum
				ent := server.portPropertyMap[portNum]
				ent.Name = bulkInfo.PortStateList[i].Name
				ent.VlanId = 0
				ent.VlanName = ""
				server.portPropertyMap[portNum] = ent
			}
			if more == false {
				return
			}
		}
	}
}
