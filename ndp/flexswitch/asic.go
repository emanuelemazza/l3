//
//Copyright [2016] [SnapRoute Inc]
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//	 Unless required by applicable law or agreed to in writing, software
//	 distributed under the License is distributed on an "AS IS" BASIS,
//	 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//	 See the License for the specific language governing permissions and
//	 limitations under the License.
//
// _______  __       __________   ___      _______.____    __    ____  __  .___________.  ______  __    __
// |   ____||  |     |   ____\  \ /  /     /       |\   \  /  \  /   / |  | |           | /      ||  |  |  |
// |  |__   |  |     |  |__   \  V  /     |   (----` \   \/    \/   /  |  | `---|  |----`|  ,----'|  |__|  |
// |   __|  |  |     |   __|   >   <       \   \      \            /   |  |     |  |     |  |     |   __   |
// |  |     |  `----.|  |____ /  .  \  .----)   |      \    /\    /    |  |     |  |     |  `----.|  |  |  |
// |__|     |_______||_______/__/ \__\ |_______/        \__/  \__/     |__|     |__|      \______||__|  |__|
//
package flexswitch

import (
	"asicd/asicdCommonDefs"
	"l3/ndp/api"
	"l3/ndp/config"
	"sync"
	"utils/commonDefs"
)

var switchInst *commonDefs.AsicdClientStruct = nil
var once sync.Once

func initAsicdNotification() commonDefs.AsicdNotification {
	nMap := make(commonDefs.AsicdNotification)
	nMap = commonDefs.AsicdNotification{
		commonDefs.NOTIFY_L2INTF_STATE_CHANGE:       true,
		commonDefs.NOTIFY_L3INTF_STATE_CHANGE:       true,
		commonDefs.NOTIFY_VLAN_CREATE:               true,
		commonDefs.NOTIFY_VLAN_DELETE:               true,
		commonDefs.NOTIFY_VLAN_UPDATE:               true,
		commonDefs.NOTIFY_LOGICAL_INTF_CREATE:       false,
		commonDefs.NOTIFY_LOGICAL_INTF_DELETE:       false,
		commonDefs.NOTIFY_LOGICAL_INTF_UPDATE:       true,
		commonDefs.NOTIFY_IPV4INTF_CREATE:           false,
		commonDefs.NOTIFY_IPV4INTF_DELETE:           false,
		commonDefs.NOTIFY_IPV6INTF_CREATE:           true,
		commonDefs.NOTIFY_IPV6INTF_DELETE:           true,
		commonDefs.NOTIFY_LAG_CREATE:                true,
		commonDefs.NOTIFY_LAG_DELETE:                true,
		commonDefs.NOTIFY_LAG_UPDATE:                true,
		commonDefs.NOTIFY_IPV4NBR_MAC_MOVE:          false,
		commonDefs.NOTIFY_IPV4_ROUTE_CREATE_FAILURE: false,
		commonDefs.NOTIFY_IPV4_ROUTE_DELETE_FAILURE: false,
	}
	return nMap
}

func GetSwitchInst() *commonDefs.AsicdClientStruct {
	once.Do(func() {
		notifyMap := initAsicdNotification()
		notifyHdl := &AsicNotificationHdl{}
		switchInst = &commonDefs.AsicdClientStruct{
			NHdl: notifyHdl,
			NMap: notifyMap,
		}
	})
	return switchInst
}

func (notifyHdl *AsicNotificationHdl) ProcessNotification(msg commonDefs.AsicdNotifyMsg) {
	switch msg.(type) {
	case commonDefs.L2IntfStateNotifyMsg:
		l2Msg := msg.(commonDefs.L2IntfStateNotifyMsg)
		if l2Msg.IfState == asicdCommonDefs.INTF_STATE_UP {
			api.SendL2PortNotification(l2Msg.IfIndex, config.STATE_UP)
		} else {
			api.SendL2PortNotification(l2Msg.IfIndex, config.STATE_DOWN)
		}
	case commonDefs.L3IntfStateNotifyMsg:
		l3Msg := msg.(commonDefs.L3IntfStateNotifyMsg)
		if l3Msg.IfState == asicdCommonDefs.INTF_STATE_UP {
			api.SendL3PortNotification(l3Msg.IfIndex, config.STATE_UP, l3Msg.IpAddr)
		} else {
			api.SendL3PortNotification(l3Msg.IfIndex, config.STATE_DOWN, l3Msg.IpAddr)
		}
	case commonDefs.VlanNotifyMsg:
		/*
			type VlanNotifyMsg struct {
				MsgType    uint8
				VlanId     uint16
				VlanName   string
				TagPorts   []int32
				UntagPorts []int32
			}
		*/
		vlanMsg := msg.(commonDefs.VlanNotifyMsg)
		oper := ""
		switch vlanMsg.MsgType {
		case commonDefs.NOTIFY_VLAN_CREATE:
			oper = config.CONFIG_CREATE
		case commonDefs.NOTIFY_VLAN_DELETE:
			oper = config.CONFIG_DELETE
		case commonDefs.NOTIFY_VLAN_UPDATE:
			oper = config.CONFIG_UPDATE
		}
		api.SendVlanNotification(oper, int32(vlanMsg.VlanId), vlanMsg.VlanName, vlanMsg.UntagPorts)
	case commonDefs.IPv6IntfNotifyMsg:
		ipv6Msg := msg.(commonDefs.IPv6IntfNotifyMsg)
		if ipv6Msg.MsgType == commonDefs.NOTIFY_IPV6INTF_CREATE {
			api.SendIPIntfNotfication(ipv6Msg.IfIndex, ipv6Msg.IpAddr, config.CONFIG_CREATE)
		} else {
			api.SendIPIntfNotfication(ipv6Msg.IfIndex, ipv6Msg.IpAddr, config.CONFIG_DELETE)
		}
	}
}
