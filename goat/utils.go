package goat

import (
    "fmt"
    "strconv"
    "net"
    "strings"
    "bufio"
    "time"
)

func itoa(n int) string {
    return fmt.Sprintf("%d", n)
}

func atoi(s string) int {
    n, _ := strconv.Atoi(s)
    return n
}

func sendToAddress(address netAddress, tokens... string) {
    sendTo(address.String(), tokens...)
}
    
func sendTo(address string, tokens... string) {
    escTokens := make([]string, len(tokens))
    for i, tok:= range tokens {
        escTokens[i] = escape(tok)
    }
    conn, err := net.Dial("tcp", address)
    if err == nil{
        fmt.Fprintf(conn, "%s\n", strings.Join(escTokens," "))
    }
}   

func listenToRandomPort() (net.Listener, int){
    return ltp(0)
}

func listenToPort(port int) net.Listener{
    lst, _ := ltp(port)
    return lst
}

func ltp(num int) (net.Listener, int){
    listener, err := net.Listen("tcp", ":"+itoa(num))
    if err != nil{
        panic(err)
    }
    //myAddressPort := listener.Addr().String()
    //portIndex := strings.LastIndex(myAddressPort, ":")
    //listeningPort := atoi(myAddressPort[portIndex+1:])
    listeningPort := atoi(newNetAddress(listener.Addr().String()).Port)
    return listener, listeningPort
}

func escapeWithType(x interface{}, isXAttr bool) string{
    if isXAttr {
        return escape("A("+x.(string)+")")
    } else {
        switch val := x.(type) {
            case string:
                return escape("S("+val+")")
            case int:
                return escape("I("+itoa(val)+")")
            case bool:
                if val {
                    return escape("B(true)")
                } else {
                    return escape("B(false)")
                }
            default: //TODO gob!
                return escape("X")
        }
    }
}

func unescapeWithType(s string, from int) (interface{}, bool, int) {
    switch s[from] {
        case 'A':
        {
            atName, next := unescape(s, from+2)
            return atName, true, next
        }
        case 'S':
        {
            str, next := unescape(s, from+2)
            return str, false, next
        }
        case 'I':
        {
            nbr, next := unescape(s, from+2)
            return atoi(nbr), false, next
        }
        case 'B':
        {
            bval, next := unescape(s, from+2)
            switch bval{
                case "true":
                    return true, false, next
                case "false":
                    return false, false, next
                default:
                    panic(bval+" is not a valid boolean!")
            }
        }
        case 'X': //TODO gob!
        {
            return nil, false, from+1
        }
        default:
            panic(s[from:]+": invalid value or attribute!")
    }
}

func toValue(attr *Attributes, x interface{}, isXAttr bool) (interface{}, bool){
    if isXAttr {
        return (*attr).Get(x.(string))
    } else {
        return x, true
    }
}

func escape(s string) string {
    rpl := strings.NewReplacer("\\","\\\\"," ","\\_",",","\\,",")","\\)","\n","\\n")
    return rpl.Replace(s)
}
func unescape(s string, from int) (string, int) {
    out := ""
    escapeRun := false
    i:=from
    for ; i<len(s); i++ {
        if escapeRun {
            switch s[i] {
                case '_':
                    out += " "
                case 'n':
                    out += "\n"
                case '\\', ',', ')':
                    out += string(s[i:i+1])
                default:
                    // TODO error!
                    out += "\\" + string(s[i:i+1])
            }
            escapeRun = false
        } else {
            switch s[i] {
                case '\\':
                    escapeRun = true
                case ',', ')':
                    return out, i
                default:
                    out += string(s[i:i+1])
            }
        }
    }
    if escapeRun {
        out += "\\"
    }
    return out, i
} 

func receive(listener net.Listener) (string, []string) {
    var to bool
    cmd, params, _ := receiveWithAddressTimeout(listener, 0, &to)
    return cmd, params
}

func receiveWithTimeout(listener net.Listener, msec int64, timedOut *bool) (string, []string) {
    cmd, params, _ := receiveWithAddressTimeout(listener, msec, timedOut)
    return cmd, params
}

func receiveWithAddress(listener net.Listener) (string, []string, netAddress) {
    var to bool
    cmd, params, addr := receiveWithAddressTimeout(listener, 0, &to)
    return cmd, params, addr
}

func receiveWithAddressTimeout(listener net.Listener, msec int64, timedOut *bool) (string, []string, netAddress) {
    var conn net.Conn
    var err error
    chnAccepted := make(chan struct{})
    var chnTimeout <-chan time.Time
    go func(){
        conn, err = listener.Accept()
        close(chnAccepted)
    }()
    if(msec > 0){
        chnTimeout = time.After(time.Duration(msec) * time.Millisecond)
    } else {
        chnTimeout = make(chan time.Time)
    }
    select{
        case <- chnAccepted:
            *timedOut = false
        case <- chnTimeout:
            *timedOut = true
            return "", []string{}, netAddress{}
    }
    if err != nil {
        panic(err)
    }
    serverMsg, err := bufio.NewReader(conn).ReadString('\n')
    if err == nil {
        escTokens := strings.Split(serverMsg[:len(serverMsg)-1], " ")
        tokens := make([]string, len(escTokens))
        for i, escTok := range escTokens {
            tokens[i], _ = unescape(escTok, 0)
        }
        return tokens[0], tokens[1:], newNetAddress(conn.RemoteAddr().String())
    } else {
        panic(err)
    }
}

type netAddress struct{
    Host string
    Port string
}
func (na *netAddress) String() string {
    return net.JoinHostPort(na.Host, na.Port)
}
func newNetAddress(addr string) netAddress{
    na := netAddress{}
    na.Host, na.Port, _ = net.SplitHostPort(addr)
    return na
}

func ToString(x interface{}) string {
    switch itm := x.(type){
        case int:
            return goat.itoa(itm)
        case bool:
            if itm {
                return "true"
            } else {
                return "false"
            }
        case string:
            return itm
        default:
            return "interface{}"
    }
}
