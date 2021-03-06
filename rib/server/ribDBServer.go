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

// ribDBServer.go
package server

import (
	"errors"
	"fmt"
	"l3/rib/ribdCommonDefs"
	"models/objects"
	"ribd"
	"strconv"
)

var dbRouteReqs []RIBdServerConfig

type RouteDBInfo struct {
	entry     RouteInfoRecord
	routeList RouteInfoRecordList
}

func (m RIBDServer) WriteIPv4RouteStateEntryToDB(dbInfo RouteDBInfo) error {
	//	logger.Info("WriteIPv4RouteStateEntryToDB")
	entry := dbInfo.entry
	routeList := dbInfo.routeList
	m.DelIPv4RouteStateEntryFromDB(dbInfo)
	var dbObj objects.IPv4RouteState
	obj := ribd.NewIPv4RouteState()
	obj.DestinationNw = entry.networkAddr
	obj.Protocol = routeList.selectedRouteProtocol //ReverseRouteProtoTypeMapDB[int(entry.protocol)]
	obj.NextHopList = make([]*ribd.NextHopInfo, 0)
	routeInfoList := routeList.routeInfoProtocolMap[routeList.selectedRouteProtocol]
	//	logger.Info("len of routeInfoList - ", len(routeInfoList), "selected route protocol = ", routeList.selectedRouteProtocol, " route Protocol: ", entry.protocol, " route nwAddr: ", entry.networkAddr)
	nextHopInfo := make([]ribd.NextHopInfo, len(routeInfoList))
	i := 0
	for sel := 0; sel < len(routeInfoList); sel++ {
		//	logger.Info("nextHop ", sel, " weight = ", routeInfoList[sel].weight, " ip ", routeInfoList[sel].nextHopIp, " intref ", routeInfoList[sel].nextHopIfIndex)
		nextHopInfo[i].NextHopIp = routeInfoList[sel].nextHopIp.String()
		nextHopInfo[i].NextHopIntRef = strconv.Itoa(int(routeInfoList[sel].nextHopIfIndex))
		intfEntry, ok := IntfIdNameMap[int32(routeInfoList[sel].nextHopIfIndex)]
		if ok {
			//	logger.Debug("Map foud for ifndex : ", routeInfoList[sel].nextHopIfIndex, "Name = ", intfEntry.name)
			nextHopInfo[i].NextHopIntRef = intfEntry.name
		}
		//logger.Debug("IntfRef = ", nextHopInfo[i].NextHopIntRef)
		nextHopInfo[i].Weight = int32(routeInfoList[sel].weight)
		obj.NextHopList = append(obj.NextHopList, &nextHopInfo[i])
		if routeInfoList[sel].protocol == int8(RouteProtocolTypeMapDB[routeList.selectedRouteProtocol]) {
			obj.IsNetworkReachable = routeInfoList[sel].resolvedNextHopIpIntf.IsReachable
		}
		i++
	}
	obj.RouteCreatedTime = entry.routeCreatedTime
	obj.RouteUpdatedTime = entry.routeUpdatedTime
	obj.PolicyList = make([]string, 0)
	routePolicyListInfo := ""
	if routeList.policyList != nil {
		for k := 0; k < len(routeList.policyList); k++ {
			routePolicyListInfo = "policy " + routeList.policyList[k] + "["
			policyRouteIndex := PolicyRouteIndex{destNetIP: entry.networkAddr, policy: routeList.policyList[k]}
			policyStmtMap, ok := PolicyEngineDB.PolicyEntityMap[policyRouteIndex]
			if !ok || policyStmtMap.PolicyStmtMap == nil {
				continue
			}
			routePolicyListInfo = routePolicyListInfo + " stmtlist[["
			for stmt, conditionsAndActionsList := range policyStmtMap.PolicyStmtMap {
				routePolicyListInfo = routePolicyListInfo + stmt + ":[conditions:"
				for c := 0; c < len(conditionsAndActionsList.ConditionList); c++ {
					routePolicyListInfo = routePolicyListInfo + conditionsAndActionsList.ConditionList[c].Name + ","
				}
				routePolicyListInfo = routePolicyListInfo + "],[actions:"
				for a := 0; a < len(conditionsAndActionsList.ActionList); a++ {
					routePolicyListInfo = routePolicyListInfo + conditionsAndActionsList.ActionList[a].Name + ","
				}
				routePolicyListInfo = routePolicyListInfo + "]]"
			}
			routePolicyListInfo = routePolicyListInfo + "]"
			obj.PolicyList = append(obj.PolicyList, routePolicyListInfo)
		}
	}
	objects.ConvertThriftToribdIPv4RouteStateObj(obj, &dbObj)
	err := dbObj.StoreObjectInDb(m.DbHdl)
	if err != nil {
		logger.Err("Failed to store IPv4RouteState entry in DB, err - ", err)
		return errors.New(fmt.Sprintln("Failed to add IPv4RouteState db : ", entry))
	}
	//logger.Info("returned successfully after write to DB for IPv4RouteState")
	return nil
}

func (m RIBDServer) WriteIPv6RouteStateEntryToDB(dbInfo RouteDBInfo) error {
	//logger.Info("WriteIPv6RouteStateEntryToDB")
	entry := dbInfo.entry
	routeList := dbInfo.routeList
	m.DelIPv6RouteStateEntryFromDB(dbInfo)
	var dbObj objects.IPv6RouteState
	obj := ribd.NewIPv6RouteState()
	obj.DestinationNw = entry.networkAddr
	obj.Protocol = routeList.selectedRouteProtocol //ReverseRouteProtoTypeMapDB[int(entry.protocol)]
	obj.NextHopList = make([]*ribd.NextHopInfo, 0)
	routeInfoList := routeList.routeInfoProtocolMap[routeList.selectedRouteProtocol]
	//logger.Info("len of routeInfoList - ", len(routeInfoList), "selected route protocol = ", routeList.selectedRouteProtocol, " route Protocol: ", entry.protocol, " route nwAddr: ", entry.networkAddr)
	nextHopInfo := make([]ribd.NextHopInfo, len(routeInfoList))
	i := 0
	for sel := 0; sel < len(routeInfoList); sel++ {
		//logger.Info("nextHop ", sel, " weight = ", routeInfoList[sel].weight, " ip ", routeInfoList[sel].nextHopIp, " intref ", routeInfoList[sel].nextHopIfIndex)
		nextHopInfo[i].NextHopIp = routeInfoList[sel].nextHopIp.String()
		nextHopInfo[i].NextHopIntRef = strconv.Itoa(int(routeInfoList[sel].nextHopIfIndex))
		intfEntry, ok := IntfIdNameMap[int32(routeInfoList[sel].nextHopIfIndex)]
		if ok {
			//logger.Debug("Map foud for ifndex : ", routeInfoList[sel].nextHopIfIndex, "Name = ", intfEntry.name)
			nextHopInfo[i].NextHopIntRef = intfEntry.name
		}
		//logger.Debug("IntfRef = ", nextHopInfo[i].NextHopIntRef)
		nextHopInfo[i].Weight = int32(routeInfoList[sel].weight)
		obj.NextHopList = append(obj.NextHopList, &nextHopInfo[i])
		if routeInfoList[sel].protocol == int8(RouteProtocolTypeMapDB[routeList.selectedRouteProtocol]) {
			obj.IsNetworkReachable = routeInfoList[sel].resolvedNextHopIpIntf.IsReachable
		}
		i++
	}
	obj.RouteCreatedTime = entry.routeCreatedTime
	obj.RouteUpdatedTime = entry.routeUpdatedTime
	obj.PolicyList = make([]string, 0)
	routePolicyListInfo := ""
	if routeList.policyList != nil {
		for k := 0; k < len(routeList.policyList); k++ {
			routePolicyListInfo = "policy " + routeList.policyList[k] + "["
			policyRouteIndex := PolicyRouteIndex{destNetIP: entry.networkAddr, policy: routeList.policyList[k]}
			policyStmtMap, ok := PolicyEngineDB.PolicyEntityMap[policyRouteIndex]
			if !ok || policyStmtMap.PolicyStmtMap == nil {
				continue
			}
			routePolicyListInfo = routePolicyListInfo + " stmtlist[["
			for stmt, conditionsAndActionsList := range policyStmtMap.PolicyStmtMap {
				routePolicyListInfo = routePolicyListInfo + stmt + ":[conditions:"
				for c := 0; c < len(conditionsAndActionsList.ConditionList); c++ {
					routePolicyListInfo = routePolicyListInfo + conditionsAndActionsList.ConditionList[c].Name + ","
				}
				routePolicyListInfo = routePolicyListInfo + "],[actions:"
				for a := 0; a < len(conditionsAndActionsList.ActionList); a++ {
					routePolicyListInfo = routePolicyListInfo + conditionsAndActionsList.ActionList[a].Name + ","
				}
				routePolicyListInfo = routePolicyListInfo + "]]"
			}
			routePolicyListInfo = routePolicyListInfo + "]"
			obj.PolicyList = append(obj.PolicyList, routePolicyListInfo)
		}
	}
	objects.ConvertThriftToribdIPv6RouteStateObj(obj, &dbObj)
	err := dbObj.StoreObjectInDb(m.DbHdl)
	if err != nil {
		//logger.Err("Failed to store IPv6RouteState entry in DB, err - ", err)
		return errors.New(fmt.Sprintln("Failed to add IPv6RouteState db : ", entry))
	}
	//logger.Info("returned successfully after write to DB for IPv6RouteState")
	return nil
}

func (m RIBDServer) DelIPv4RouteStateEntryFromDB(dbInfo RouteDBInfo) error {
	//logger.Info("DelIPv4RouteStateEntryFromDB")
	entry := dbInfo.entry
	var dbObj objects.IPv4RouteState
	obj := ribd.NewIPv4RouteState()
	obj.DestinationNw = entry.networkAddr
	objects.ConvertThriftToribdIPv4RouteStateObj(obj, &dbObj)
	err := dbObj.DeleteObjectFromDb(m.DbHdl)
	if err != nil {
		return errors.New(fmt.Sprintln("Failed to delete IPv4RouteState from state db : ", entry))
	}
	return nil
}

func (m RIBDServer) DelIPv6RouteStateEntryFromDB(dbInfo RouteDBInfo) error {
	//logger.Info("DelIPv6RouteStateEntryFromDB")
	entry := dbInfo.entry
	var dbObj objects.IPv6RouteState
	obj := ribd.NewIPv6RouteState()
	obj.DestinationNw = entry.networkAddr
	objects.ConvertThriftToribdIPv6RouteStateObj(obj, &dbObj)
	err := dbObj.DeleteObjectFromDb(m.DbHdl)
	if err != nil {
		return errors.New(fmt.Sprintln("Failed to delete IPv6RouteState from state db : ", entry))
	}
	return nil
}

func (m RIBDServer) ReadAndUpdateRoutesFromDB() {
	//logger.Debug("ReadAndUpdateRoutesFromDB")
	var dbObjCfg objects.IPv4Route
	objList, err := m.DbHdl.GetAllObjFromDb(dbObjCfg)
	if err == nil {
		//logger.Debug("Number of routes from DB: ", len((objList)))
		for idx := 0; idx < len(objList); idx++ {
			obj := ribd.NewIPv4Route()
			dbObj := objList[idx].(objects.IPv4Route)
			objects.ConvertribdIPv4RouteObjToThrift(&dbObj, obj)
			err = m.RouteConfigValidationCheck(obj, "add")
			if err != nil {
				logger.Err("Route validation failed when reading from db")
				continue
			}
			m.RouteConfCh <- RIBdServerConfig{
				OrigConfigObject: obj,
				Op:               "add",
			}
			/*rv, _ := ribdServiceHandler.ProcessRouteCreateConfig(obj)
			if rv == false {
				//logger.Err("IPv4Route create failed during init")
			}*/
		}
	} else {
		logger.Err("DB Query failed during IPv4Route query: RIBd init")
	}
}
func (m RIBDServer) ReadAndUpdatev6RoutesFromDB() {
	//logger.Debug("ReadAndUpdatev6RoutesFromDB")
	var dbObjCfg objects.IPv6Route
	objList, err := m.DbHdl.GetAllObjFromDb(dbObjCfg)
	if err == nil {
		//logger.Debug("Number of v6 routes from DB: ", len((objList)))
		for idx := 0; idx < len(objList); idx++ {
			obj := ribd.NewIPv6Route()
			dbObj := objList[idx].(objects.IPv6Route)
			objects.ConvertribdIPv6RouteObjToThrift(&dbObj, obj)
			err = m.IPv6RouteConfigValidationCheck(obj, "add")
			if err != nil {
				logger.Err("Route validation failed when reading from db")
				continue
			}
			m.RouteConfCh <- RIBdServerConfig{
				OrigConfigObject: obj,
				Op:               "addv6",
			}
		}
	} else {
		logger.Err("DB Query failed during IPv6Route query: RIBd init")
	}
}
func (ribdServiceHandler *RIBDServer) StartDBServer() {
	logger.Info("Starting the DB update server loop")
	for {
		select {
		case info := <-ribdServiceHandler.DBRouteCh:
			if info.Op == "add" {
				dbInfo := info.OrigConfigObject.(RouteDBInfo)
				routeList := dbInfo.routeList
				routeInfoList := routeList.routeInfoProtocolMap[routeList.selectedRouteProtocol]
				for sel := 0; sel < len(routeInfoList); sel++ {
					if routeInfoList[sel].protocol == int8(RouteProtocolTypeMapDB[routeList.selectedRouteProtocol]) {
						//logger.Debug("add case iptype = ", routeInfoList[sel].ipType)
						if routeInfoList[sel].ipType == ribdCommonDefs.IPv6 {
							info.Op = "addv6"
						}
					}
				}
			} else if info.Op == "del" {
				//logger.Debug("MADHAVI!! del case")
				dbInfo := info.OrigConfigObject.(RouteDBInfo)
				entry := dbInfo.entry
				//logger.Debug("del case iptype = ", entry.ipType)
				if entry.ipType == ribdCommonDefs.IPv6 {
					info.Op = "delv6"
				}
			}
			//logger.Info(" received message on DBRouteCh, op:", info.Op)
			if info.Op == "add" {
				ribdServiceHandler.WriteIPv4RouteStateEntryToDB(info.OrigConfigObject.(RouteDBInfo))
			} else if info.Op == "addv6" {
				ribdServiceHandler.WriteIPv6RouteStateEntryToDB(info.OrigConfigObject.(RouteDBInfo))
			} else if info.Op == "del" {
				ribdServiceHandler.DelIPv4RouteStateEntryFromDB(info.OrigConfigObject.(RouteDBInfo))
			} else if info.Op == "delv6" {
				ribdServiceHandler.DelIPv6RouteStateEntryFromDB(info.OrigConfigObject.(RouteDBInfo))
			} else if info.Op == "fetch" {
				ribdServiceHandler.ReadAndUpdateRoutesFromDB()
				ribdServiceHandler.ReadAndUpdatev6RoutesFromDB()
				logger.Info("Signalling dbread to be true")
				ribdServiceHandler.DBReadDone <- true
			}
		}
	}
}

/*
func (ribdServiceHandler *RIBDServer) StartDBServer() {
	//logger.Info("Starting the DB update server loop with checkDBReq()")
	for {
		//logger.Info(fmt.Sprintln("for loop beginning dbReqCount", dbReqCount))
		select {
		case dbRouteInfo := <-ribdServiceHandler.DBRouteCh:
			if dbRouteInfo.Op != "fetch" {
				//logger.Info(fmt.Sprintln("Not a fetch case, op:", dbRouteInfo.Op))
				if dbReqCount == 0 {
					dbRouteReqs = make([]RIBdServerConfig, dbReqCount)
				}
				dbReqCount++
				dbRouteReqs = append(dbRouteReqs, dbRouteInfo)
				//logger.Info(fmt.Sprintln("dbReqCount", dbReqCount))
				if dbReqCount < dbReqCountLimit {
					dbReqCheckCountLimit++
				} else {
					//logger.Info(fmt.Sprintln("process dbRouteReqs of len:", len(dbRouteReqs)))
					for idx := 0; idx < len(dbRouteReqs); idx++ {
						//logger.Info(fmt.Sprintln("process dbRouteReq idx :", idx))
						info := dbRouteReqs[idx]
						if info.Op == "add" {
							dbInfo := info.OrigConfigObject.(RouteDBInfo)
							routeList := dbInfo.routeList
							routeInfoList := routeList.routeInfoProtocolMap[routeList.selectedRouteProtocol]
							for sel := 0; sel < len(routeInfoList); sel++ {
								if routeInfoList[sel].protocol == int8(RouteProtocolTypeMapDB[routeList.selectedRouteProtocol]) {
									//logger.Debug(fmt.Sprintln("add case iptype = ", routeInfoList[sel].ipType))
									if routeInfoList[sel].ipType == ribdCommonDefs.IPv6 {
										info.Op = "addv6"
									}
								}
							}
						} else if info.Op == "del" {
							dbInfo := info.OrigConfigObject.(RouteDBInfo)
							entry := dbInfo.entry
							if entry.ipType == ribdCommonDefs.IPv6 {
								info.Op = "delv6"
							}
						}
						//logger.Info(fmt.Sprintln(" received message on DBRouteCh, op:", info.Op))
						if info.Op == "add" {
							ribdServiceHandler.WriteIPv4RouteStateEntryToDB(info.OrigConfigObject.(RouteDBInfo))
						} else if info.Op == "addv6" {
							ribdServiceHandler.WriteIPv6RouteStateEntryToDB(info.OrigConfigObject.(RouteDBInfo))
						} else if info.Op == "del" {
							ribdServiceHandler.DelIPv4RouteStateEntryFromDB(info.OrigConfigObject.(RouteDBInfo))
						} else if info.Op == "delv6" {
							ribdServiceHandler.DelIPv6RouteStateEntryFromDB(info.OrigConfigObject.(RouteDBInfo))
						}
					}
					dbReqCount = 0
					dbReqCheckCount = 0
					dbRouteReqs = nil
					//logger.Info("else case  - cleared counters")
				}

			} else { //if dbRouteinfo.Op == "fetch" {
				//logger.Info(fmt.Sprintln("fetch case, dbReqCount:", dbReqCount))
				ribdServiceHandler.ReadAndUpdateRoutesFromDB()
				ribdServiceHandler.ReadAndUpdatev6RoutesFromDB()
				//logger.Debug(fmt.Sprintln("Signalling dbread to be true"))
				ribdServiceHandler.DBReadDone <- true
			}
		}
	}
}
*/
