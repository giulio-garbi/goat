package goat

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

type CentralServer struct {
	nextCompId           int
	nextMsgId            int
	compAddresses        map[int]string
	listener             net.Listener
	messagesActuallySent int
	proposalsActuallySent int
}

func (srv *CentralServer) sendToComponent(cid int, tokens ...string) {
	escTokens := make([]string, len(tokens))
	for i, tok := range tokens {
		escTokens[i] = escape(tok)
	}
	for successComm := false; !successComm; {
		conn, err := net.Dial("tcp", srv.compAddresses[cid])
		if err == nil {
			fmt.Fprintf(conn, "%s\n", strings.Join(escTokens, " "))
			successComm = true
		}
	}
}

func (srv *CentralServer) Terminate() {
	srv.listener.Close()

}

func (srv *CentralServer) receive() (string, []string, string) {
	conn, err := srv.listener.Accept()
	if err != nil {
		//TODO Error
		return "", []string{}, ""
	}
	myAddressPort := conn.RemoteAddr().String()
	portIndex := strings.LastIndex(myAddressPort, ":")
	address := myAddressPort[:portIndex]
	_ = err
	serverMsg, err := bufio.NewReader(conn).ReadString('\n')
	if err == nil {
		escTokens := strings.Split(serverMsg[:len(serverMsg)-1], " ")
		tokens := make([]string, len(escTokens))
		for i, escTok := range escTokens {
			tokens[i], _ = unescape(escTok, 0)
		}
		return tokens[0], tokens[1:], address
	} else {
		//TODO Error
		return "", []string{}, ""
	}
}

func (srv *CentralServer) GetMessagesSent() int {
	return srv.messagesActuallySent
}

func (srv *CentralServer) GetProposalsSent() int {
	return srv.proposalsActuallySent
}

func RunCentralServer(port int, term chan struct{}, msec int64) *CentralServer {
	srv := CentralServer{
		nextCompId:           0,
		nextMsgId:            0,
		compAddresses:        map[int]string{},
		messagesActuallySent: 0,
		proposalsActuallySent: 0,
	}
	srv.listener, _ = net.Listen("tcp", ":"+itoa(port))
	go func() {
		for {
			ok := make(chan struct{})
			var cmd string
			var params []string
			var address string
			go func() {
				cmd, params, address = srv.receive()
				close(ok)
			}()
			select {
			case <-ok:
			case <-timeout(msec):
				close(term)
				return
			}
			switch cmd {
			case "Register":
				cPort := params[0]
				cid := srv.nextCompId
				srv.nextCompId++
				srv.compAddresses[cid] = address + ":" + cPort
				srv.sendToComponent(cid, "Registered", itoa(cid), itoa(srv.nextMsgId))
			case "DATA":
				senderid := atoi(params[1])
				if params[2] != "FF" {
					srv.messagesActuallySent++
					tuple := decodeTuple(params[3])
					if tuple.Get(0) == "propose" {
					    srv.proposalsActuallySent++
					}
				}
				for cid := range srv.compAddresses {
					if senderid != cid {
						srv.sendToComponent(cid, append([]string{"DATA"}, params...)...)
					}
				}
			case "REQ":
				cid := atoi(params[0])
				mid := srv.nextMsgId
				srv.nextMsgId++
				srv.sendToComponent(cid, "RPLY", itoa(mid))
			}
		}
	}()
	return &srv
}
