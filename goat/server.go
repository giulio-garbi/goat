package goat

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
)

type CentralServer struct {
	nextCompId           int
	nextMsgId            int
	//compAddresses        map[int]string
	listener             net.Listener
	messagesExchanged    int
	lock *sync.Mutex
	compConnOut map[int]net.Conn
	compConnIn map[int]*bufio.Reader
}

func (srv *CentralServer) sendToComponent(cid int, tokens ...string) {
	escTokens := make([]string, len(tokens))
	for i, tok := range tokens {
		escTokens[i] = escape(tok)
	}
	//dprintln("Dialing", cid)
	for successComm := false; !successComm; {
		conn := srv.compConnOut[cid]//, err := net.Dial("tcp", srv.compAddresses[cid])
//		if err == nil {
		    dprintln("Dialed", cid)
	        dprintln("Writing", cid)
			_, errf := fmt.Fprintf(conn, "%s\n", strings.Join(escTokens, " "))
			if errf != nil {
			    panic(errf)
			}
	        dprintln("Written", cid)
			srv.messagesExchanged++
			successComm = true
	//	} else {
		//    panic(err)
		//} 
	}
}

func (srv *CentralServer) Terminate() {
	srv.listener.Close()

}
/*
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
	    dprintln("Accept:",serverMsg)
		escTokens := strings.Split(serverMsg[:len(serverMsg)-1], " ")
		tokens := make([]string, len(escTokens))
		for i, escTok := range escTokens {
			tokens[i], _ = unescape(escTok, 0)
		}
		srv.messagesExchanged++
		return tokens[0], tokens[1:], address
	} else {
		//TODO Error
		return "", []string{}, ""
	}
}*/

func (srv *CentralServer) GetMessagesExchanged() int {
	return srv.messagesExchanged
}

func (srv *CentralServer) ListenReg() {
    for{
        conn, _ := srv.listener.Accept()
        bconn := bufio.NewReader(conn)
	    myAddressPort := conn.RemoteAddr().String()
	    portIndex := strings.LastIndex(myAddressPort, ":")
	    address := myAddressPort[:portIndex]
        dprintln("!")
	    serverMsg, err := bufio.NewReader(conn).ReadString('\n')
	    if err == nil {
	        dprintln("Accept:",serverMsg)
		    escTokens := strings.Split(serverMsg[:len(serverMsg)-1], " ")
		    tokens := make([]string, len(escTokens))
		    for i, escTok := range escTokens {
			    tokens[i], _ = unescape(escTok, 0)
		    }
		    cPort := tokens[1]
		    srv.lock.Lock()
		    srv.messagesExchanged++
			cid := srv.nextCompId
			srv.nextCompId++
			srv.compConnIn[cid] = bconn
			connOut, err := net.Dial("tcp", address + ":" + cPort)
			if err != nil {
			    panic(err)
			}
			srv.compConnOut[cid] = connOut
			srv.sendToComponent(cid, "Registered", itoa(cid), itoa(srv.nextMsgId))
			go func(id int, bcon *bufio.Reader){srv.ListenConn(id, bcon)}(cid, bconn)
		    srv.lock.Unlock()
	    } 
    }
}

func (srv *CentralServer) ListenConn(cid int, bconn *bufio.Reader) {
    for{
        serverMsg, _ := bconn.ReadString('\n')
        if serverMsg == "" {
            continue
        }
        dprintln("Accept:",serverMsg)
	    escTokens := strings.Split(serverMsg[:len(serverMsg)-1], " ")
	    tokens := make([]string, len(escTokens))
	    for i, escTok := range escTokens {
		    tokens[i], _ = unescape(escTok, 0)
	    }
	    params := tokens[1:]
	    srv.lock.Lock()
	    srv.messagesExchanged++
	    switch(tokens[0]) {
	        case "DATA":
				senderid := atoi(params[1])
				for cid := range srv.compConnOut {
					if senderid != cid {
					    dprintln("Sending msg to",cid,params)
						srv.sendToComponent(cid, append([]string{"DATA"}, params...)...)
						dprintln("Sent msg to",cid,params, srv.nextMsgId)
					} else {
					    dprintln("Skipping msg to",cid,params)
					}
				}
			case "REQ":
				cid := atoi(params[0])
				mid := srv.nextMsgId
				srv.nextMsgId++
				dprintln("Sending RPLY to",cid)
				srv.sendToComponent(cid, "RPLY", itoa(mid))
			
		}
		srv.lock.Unlock()
    }
}

func RunCentralServer(port int, term chan struct{}, msec int64) *CentralServer {
	srv := CentralServer{
		nextCompId:           0,
		nextMsgId:            0,
		//compAddresses:        map[int]string{},
		messagesExchanged:    0,
	    lock: &sync.Mutex{},
	    compConnOut: map[int]net.Conn{},
	    compConnIn: map[int]*bufio.Reader{},
	}
	var err error
	srv.listener, err = net.Listen("tcp", ":"+itoa(port))
	if err != nil{
	    panic(err)
	}
	go func() {
	    srv.ListenReg()
		/*for {
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
				for cid := range srv.compAddresses {
					if senderid != cid {
					    fmt.Println("Sending msg to",cid, srv.compAddresses[cid],params)
						srv.sendToComponent(cid, append([]string{"DATA"}, params...)...)
						fmt.Println("Sent msg to",cid, srv.compAddresses[cid],params, srv.nextMsgId)
					} else {
					    fmt.Println("Skipping msg to",cid, srv.compAddresses[cid],params)
					}
				}
			case "REQ":
				cid := atoi(params[0])
				mid := srv.nextMsgId
				srv.nextMsgId++
				dprintln("Sending RPLY to",cid, srv.compAddresses[cid])
				srv.sendToComponent(cid, "RPLY", itoa(mid))
			}
		}*/
	}()
	return &srv
}
