package goat

import (
    "net"
    "bufio"
    "sync"
    "fmt"
    "strings"
)

type duplexConn struct {
    conn net.Conn
    reader *bufio.Reader
    lock *sync.Mutex
}

func (dc *duplexConn) Send(tokens ...string) error{
    escTokens := make([]string, len(tokens))
    for i, tok:= range tokens {
        escTokens[i] = escape(tok)
    }
    dc.lock.Lock()
    _, err := fmt.Fprintf(dc.conn, "%s\n", strings.Join(escTokens," "))
    dc.lock.Unlock()
    return err
}

func (dc *duplexConn) SrcAddr() netAddress{
    return newNetAddress(dc.conn.RemoteAddr().String())
}

func (dc *duplexConn) RemoteAddr() net.Addr{
    return dc.conn.RemoteAddr()
}

func (dc *duplexConn) ReceiveErr() (string, []string, error) {
    serverMsg, err := dc.reader.ReadString('\n')
    if err == nil {
        escTokens := strings.Split(serverMsg[:len(serverMsg)-1], " ")
        tokens := make([]string, len(escTokens))
        for i, escTok := range escTokens {
            tokens[i], _ = unescape(escTok, 0)
        }
        return tokens[0], tokens[1:], nil
    } else {
        return "", nil, err
    }
}

func (dc *duplexConn) Receive() (string, []string) {
    cmd, params, err := dc.ReceiveErr()
    if err != nil {
        panic(err)
    } else {
        return cmd, params
    }
}


func (dc *duplexConn) Close() {
    dc.conn.Close()
}

func connectWith(address string) *duplexConn {
    conn, err := net.Dial("tcp", address)
    if err == nil{
        return newDuplexConn(conn)
    } else {
        panic(err)
    }
}

func newDuplexConn(conn net.Conn) *duplexConn {
    return &duplexConn{conn, bufio.NewReader(conn), &sync.Mutex{}}
}




func listenerInt(port int) (*unboundChanConn, chan struct{}, int){
    uc := newUnboundChanConn()
    chnReady := make(chan struct{})
    chnPort := make(chan int, 1)
    go func(){
        listener, err := net.Listen("tcp", ":"+itoa(port))
        if err != nil{
            panic(err)
        }
        chnPort <- atoi(newNetAddress(listener.Addr().String()).Port)
        close(chnReady)
        for{
            conn, err := listener.Accept()
            if err == nil {
                uc.In <- newDuplexConn(conn)
            }
        }
    }()
    return uc, chnReady, <-chnPort
}

func listener(port int) (*unboundChanConn, chan struct{}) {
    ucc, rd, _ := listenerInt(port)
    return ucc, rd
}


func listenerRandomPort() (*unboundChanConn, chan struct{}, int){
    return listenerInt(0)
}
