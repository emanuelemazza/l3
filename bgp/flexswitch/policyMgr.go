package FSMgr

import (
	"bytes"
	"encoding/json"
	"fmt"
	nanomsg "github.com/op/go-nanomsg"
	"l3/bgp/api"
	"l3/rib/ribdCommonDefs"
	"models"
	"utils/logging"
)

/*  Init policy manager with specific needs
 */
func NewFSPolicyMgr(logger *logging.Writer, fileName string) *FSPolicyMgr {
	mgr := &FSPolicyMgr{
		plugin: "ovsdb",
		logger: logger,
	}

	return mgr
}

func (mgr *FSPolicyMgr) AddPolicy() {

}

func (mgr *FSPolicyMgr) RemovePolicy() {
}

/*  Start nano msg socket with ribd
 */
func (mgr *FSPolicyMgr) Start() {
	mgr.logger.Info("Starting policyMgr")
	mgr.policySubSocket, _ = mgr.setupSubSocket(ribdCommonDefs.PUB_SOCKET_POLICY_ADDR)
	go mgr.listenForPolicyUpdates(mgr.policySubSocket)
}

func (mgr *FSPolicyMgr) setupSubSocket(address string) (*nanomsg.SubSocket, error) {
	var err error
	var socket *nanomsg.SubSocket
	if socket, err = nanomsg.NewSubSocket(); err != nil {
		mgr.logger.Err(fmt.Sprintf("Failed to create subscribe socket %s",
			"error:%s", address, err))
		return nil, err
	}

	if err = socket.Subscribe(""); err != nil {
		mgr.logger.Err(fmt.Sprintf("Failed to subscribe to \"\" on ",
			"subscribe socket %s, error:%s", address, err))
		return nil, err
	}

	if _, err = socket.Connect(address); err != nil {
		mgr.logger.Err(fmt.Sprintf("Failed to connect to publisher socket %s,",
			"error:%s", address, err))
		return nil, err
	}

	mgr.logger.Info(fmt.Sprintf("Connected to publisher socker %s", address))
	if err = socket.SetRecvBuffer(1024 * 1024); err != nil {
		mgr.logger.Err(fmt.Sprintln("Failed to set the buffer size for",
			"subsriber socket %s, error:", address, err))
		return nil, err
	}
	return socket, nil
}
func (mgr *FSPolicyMgr) handlePolicyConditionUpdates(msg ribdCommonDefs.RibdNotifyMsg) {
	policyCondition := models.PolicyCondition{}
	updateMsg := "Add"
	if msg.MsgType == ribdCommonDefs.NOTIFY_POLICY_CONDITION_DELETED {
		updateMsg = "Remove"
	} else if msg.MsgType == ribdCommonDefs.NOTIFY_POLICY_CONDITION_UPDATED {
		updateMsg = "Update"
	}
	err := json.Unmarshal(msg.MsgBuf, &policyCondition)
	if err != nil {
		mgr.logger.Err(fmt.Sprintf(
			"Unmarshal RIB policy condition update failed with err %s", err))
	}
	mgr.logger.Info(fmt.Sprintln(updateMsg, "Policy Condition ", policyCondition.Name, " type: ", policyCondition.ConditionType))
	if msg.MsgType == ribdCommonDefs.NOTIFY_POLICY_CONDITION_CREATED {
		api.SendPolicyConditionNotification(&policyCondition, nil, nil)
	} else if msg.MsgType == ribdCommonDefs.NOTIFY_POLICY_CONDITION_DELETED {
		api.SendPolicyConditionNotification(nil, &policyCondition, nil)
	} else if msg.MsgType == ribdCommonDefs.NOTIFY_POLICY_CONDITION_UPDATED {
		api.SendPolicyConditionNotification(nil, nil, &policyCondition)
	}
}
func (mgr *FSPolicyMgr) handlePolicyUpdates(rxBuf []byte) {
	reader := bytes.NewReader(rxBuf)
	decoder := json.NewDecoder(reader)
	msg := ribdCommonDefs.RibdNotifyMsg{}
	err := decoder.Decode(&msg)
	if err != nil {
		mgr.logger.Err(fmt.Sprintln("Error while decoding msg"))
		return
	}
	switch msg.MsgType {
	case ribdCommonDefs.NOTIFY_POLICY_CONDITION_CREATED, ribdCommonDefs.NOTIFY_POLICY_CONDITION_DELETED,
		ribdCommonDefs.NOTIFY_POLICY_CONDITION_UPDATED:
		mgr.handlePolicyConditionUpdates(msg)
	default:
		mgr.logger.Err(fmt.Sprintf("**** Received Policy update with ",
			"unknown type %d ****", msg.MsgType))
	}
}

func (mgr *FSPolicyMgr) listenForPolicyUpdates(socket *nanomsg.SubSocket) {
	for {
		mgr.logger.Info("Read on Policy subscriber socket...")
		rxBuf, err := socket.Recv(0)
		if err != nil {
			mgr.logger.Err(fmt.Sprintln("Recv on Policy subscriber socket",
				"failed with error:", err))
			continue
		}
		mgr.logger.Info(fmt.Sprintln("Policy subscriber recv returned:", rxBuf))
		mgr.handlePolicyUpdates(rxBuf)
	}
}
