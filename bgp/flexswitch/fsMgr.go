package FSMgr

import (
	"asicdServices"
	"bfdd"
	nanomsg "github.com/op/go-nanomsg"
	"ribd"
	"utils/logging"
)

/*  Router manager will handle all the communication with ribd
 */
type FSRouteMgr struct {
	plugin          string
	logger          *logging.Writer
	ribdClient      *ribd.RIBDServicesClient
	ribSubSocket    *nanomsg.SubSocket
	ribSubBGPSocket *nanomsg.SubSocket
}

/*  Interface manager will handle all the communication with asicd
 */
type FSIntfMgr struct {
	plugin               string
	logger               *logging.Writer
	AsicdClient          *asicdServices.ASICDServicesClient
	asicdL3IntfSubSocket *nanomsg.SubSocket
}

/*  @FUTURE: this will be using in future if FlexSwitch is planning to support
 *	     daemon which is handling policy statments
 */
type FSPolicyMgr struct {
	plugin string
	logger *logging.Writer
	policySubSocket *nanomsg.SubSocket
}

/*  BFD manager will handle all the communication with bfd daemon
 */
type FSBfdMgr struct {
	plugin       string
	logger       *logging.Writer
	bfddClient   *bfdd.BFDDServicesClient
	bfdSubSocket *nanomsg.SubSocket
}

func (mgr *FSIntfMgr) PortStateChange() {

}
