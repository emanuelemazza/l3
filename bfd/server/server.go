package server

import (
	"asicd/pluginManager/pluginCommon"
	"asicdServices"
	"encoding/json"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	nanomsg "github.com/op/go-nanomsg"
	"io/ioutil"
	"l3/bfd/bfddCommonDefs"
	"log/syslog"
	"net"
	"ribd"
	"strconv"
	"time"
	"utils/ipcutils"
)

type ClientJson struct {
	Name string `json:Name`
	Port int    `json:Port`
}

type BfdClientBase struct {
	Address            string
	Transport          thrift.TTransport
	PtrProtocolFactory *thrift.TBinaryProtocolFactory
	IsConnected        bool
}

type RibdClient struct {
	BfdClientBase
	ClientHdl *ribd.RouteServiceClient
}

type IpIntfProperty struct {
	IpAddr  net.IP
	NetMask []byte
}

type BfdInterface struct {
	Enabled     bool
	NumSessions int32
	conf        IntfConfig
	property    IpIntfProperty
}

type BfdSessionMgmt struct {
	DestIp   string
	Protocol int32
}

type BfdSession struct {
	state             SessionState
	sessionTimer      *time.Timer
	txTimer           *time.Timer
	TxTimeoutCh       chan int32
	SessionTimeoutCh  chan int32
	bfdPacket         *BfdControlPacket
	SessionDeleteCh   chan bool
	pollSequence      bool
	pollSequenceFinal bool
	authEnabled       bool
	authType          AuthenticationType
	authSeqNum        uint32
	authKeyId         uint32
	authData          string
}

type BfdGlobal struct {
	Enabled              bool
	NumInterfaces        uint32
	Interfaces           map[int32]*BfdInterface
	InterfacesIdSlice    []int32
	NumSessions          uint32
	Sessions             map[int32]*BfdSession
	SessionsIdSlice      []int32
	NumUpSessions        uint32
	NumDownSessions      uint32
	NumAdminDownSessions uint32
}

type BFDServer struct {
	logger              *syslog.Writer
	ribdClient          RibdClient
	asicdClient         AsicdClient
	GlobalConfigCh      chan GlobalConfig
	IntfConfigCh        chan IntfConfig
	IntfConfigDeleteCh  chan int32
	asicdSubSocket      *nanomsg.SubSocket
	asicdSubSocketCh    chan []byte
	asicdSubSocketErrCh chan error
	portPropertyMap     map[int32]PortProperty
	vlanPropertyMap     map[int32]VlanProperty
	IPIntfPropertyMap   map[string]IPIntfProperty
	CreateSessionCh     chan BfdSessionMgmt
	DeleteSessionCh     chan BfdSessionMgmt
	AdminUpSessionCh    chan BfdSessionMgmt
	AdminDownSessionCh  chan BfdSessionMgmt
	SessionConfigCh     chan SessionConfig
	bfddPubSocket       *nanomsg.PubSocket
	bfdGlobal           BfdGlobal
}

func NewBFDServer(logger *syslog.Writer) *BFDServer {
	bfdServer := &BFDServer{}
	bfdServer.logger = logger
	bfdServer.GlobalConfigCh = make(chan GlobalConfig)
	bfdServer.IntfConfigCh = make(chan IntfConfig)
	bfdServer.IntfConfigDeleteCh = make(chan int32)
	bfdServer.asicdSubSocketCh = make(chan []byte)
	bfdServer.asicdSubSocketErrCh = make(chan error)
	bfdServer.portPropertyMap = make(map[int32]PortProperty)
	bfdServer.vlanPropertyMap = make(map[int32]VlanProperty)
	bfdServer.SessionConfigCh = make(chan SessionConfig)
	bfdServer.bfdGlobal.Enabled = false
	bfdServer.bfdGlobal.NumInterfaces = 0
	bfdServer.bfdGlobal.Interfaces = make(map[int32]*BfdInterface)
	bfdServer.bfdGlobal.InterfacesIdSlice = []int32{}
	bfdServer.bfdGlobal.NumSessions = 0
	bfdServer.bfdGlobal.Sessions = make(map[int32]*BfdSession)
	bfdServer.bfdGlobal.SessionsIdSlice = []int32{}
	bfdServer.bfdGlobal.NumUpSessions = 0
	bfdServer.bfdGlobal.NumDownSessions = 0
	bfdServer.bfdGlobal.NumAdminDownSessions = 0
	return bfdServer
}

func (server *BFDServer) ConnectToClients(paramsFile string) {
	var clientsList []ClientJson

	bytes, err := ioutil.ReadFile(paramsFile)
	if err != nil {
		server.logger.Info("Error in reading configuration file")
		return
	}

	err = json.Unmarshal(bytes, &clientsList)
	if err != nil {
		server.logger.Info("Error in Unmarshalling Json")
		return
	}

	for _, client := range clientsList {
		if client.Name == "asicd" {
			server.logger.Info(fmt.Sprintln("found asicd at port", client.Port))
			server.asicdClient.Address = "localhost:" + strconv.Itoa(client.Port)
			server.asicdClient.Transport, server.asicdClient.PtrProtocolFactory, _ = ipcutils.CreateIPCHandles(server.asicdClient.Address)
			if server.asicdClient.Transport != nil && server.asicdClient.PtrProtocolFactory != nil {
				server.logger.Info("connecting to asicd")
				server.asicdClient.ClientHdl = asicdServices.NewASICDServicesClientFactory(server.asicdClient.Transport, server.asicdClient.PtrProtocolFactory)
				server.asicdClient.IsConnected = true
			}
		} else if client.Name == "ribd" {
			server.logger.Info(fmt.Sprintln("found ribd at port", client.Port))
			server.ribdClient.Address = "localhost:" + strconv.Itoa(client.Port)
			server.ribdClient.Transport, server.ribdClient.PtrProtocolFactory, _ = ipcutils.CreateIPCHandles(server.ribdClient.Address)
			if server.ribdClient.Transport != nil && server.ribdClient.PtrProtocolFactory != nil {
				server.logger.Info("connecting to ribd")
				server.ribdClient.ClientHdl = ribd.NewRouteServiceClientFactory(server.ribdClient.Transport, server.ribdClient.PtrProtocolFactory)
				server.ribdClient.IsConnected = true
			}
		}
	}
}

func (server *BFDServer) InitPublisher(pub_str string) (pub *nanomsg.PubSocket) {
	server.logger.Info(fmt.Sprintln("Setting up ", pub_str, "publisher"))
	pub, err := nanomsg.NewPubSocket()
	if err != nil {
		server.logger.Info(fmt.Sprintln("Failed to open pub socket"))
		return nil
	}
	ep, err := pub.Bind(pub_str)
	if err != nil {
		server.logger.Info(fmt.Sprintln("Failed to bind pub socket - ", ep))
		return nil
	}
	err = pub.SetSendBuffer(1024)
	if err != nil {
		server.logger.Info(fmt.Sprintln("Failed to set send buffer size"))
		return nil
	}
	return pub
}

func (server *BFDServer) InitServer(paramFile string) {
	server.logger.Info(fmt.Sprintln("Starting Bfd Server"))
	server.ConnectToClients(paramFile)
	server.initBfdGlobalConfDefault()
	server.BuildPortPropertyMap()
	server.GetIPv4Interfaces()
	/*
		server.logger.Info("Listen for RIBd updates")
		server.listenForRIBUpdates(ribdCommonDefs.PUB_SOCKET_ADDR)
		go createRIBSubscriber()
		server.connRoutesTimer.Reset(time.Duration(10) * time.Second)
	*/
}

func (server *BFDServer) StartServer(paramFile string) {
	server.InitServer(paramFile)
	server.logger.Info("Listen for ASICd updates")
	server.listenForASICdUpdates(pluginCommon.PUB_SOCKET_ADDR)
	go server.createASICdSubscriber()
	// Start session handler
	go server.StartSessionHandler()
	// Initialize publisher
	server.bfddPubSocket = server.InitPublisher(bfddCommonDefs.PUB_SOCKET_ADDR)
	for {
		select {
		case gConf := <-server.GlobalConfigCh:
			server.logger.Info(fmt.Sprintln("Received call for performing Global Configuration", gConf))
			server.processGlobalConfig(gConf)
		case ifConf := <-server.IntfConfigCh:
			server.logger.Info(fmt.Sprintln("Received call for performing Intf Configuration", ifConf))
			server.processIntfConfig(ifConf)
		case ifIndex := <-server.IntfConfigDeleteCh:
			server.logger.Info(fmt.Sprintln("Received call for performing Intf delete", ifIndex))
			server.processIntfConfigDelete(ifIndex)
		case asicdrxBuf := <-server.asicdSubSocketCh:
			server.processAsicdNotification(asicdrxBuf)
		case <-server.asicdSubSocketErrCh:
		case sessionConfig := <-server.SessionConfigCh:
			server.logger.Info(fmt.Sprintln("Received call for performing Session Configuration", sessionConfig))
			server.processSessionConfig(sessionConfig)
			/*
				case ribrxBuf := <-server.ribSubSocketCh:
					server.processRibdNotification(ribdrxBuf)
				case <-server.connRoutesTimer.C:
					routes, _ := server.ribdClient.ClientHdl.GetConnectedRoutesInfo()
					server.logger.Info(fmt.Sprintln("Received Connected Routes:", routes))
					//server.ProcessConnectedRoutes(routes, make([]*ribd.Routes, 0))
					//server.connRoutesTimer.Reset(time.Duration(10) * time.Second)

				case <-server.ribSubSocketErrCh:
			*/
		}
	}
}
