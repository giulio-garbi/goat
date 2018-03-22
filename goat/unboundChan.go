package goat

type unboundChanUnit struct {
    In chan struct{}
    Out chan struct{}
}

func (uc *unboundChanUnit) start(){
    buffer := []struct{}{}
    for{
        for len(buffer) > 0 {
            select {
                case uc.Out <- buffer[0]:
                    buffer = buffer[1:]
                case d := <- uc.In:
                    buffer = append(buffer, d)
            }
        }
        for len(buffer) == 0 {
            d := <- uc.In
            buffer = append(buffer, d)
        }
    }
}




type unboundChanInt struct {
    In chan int
    Out chan int
}

func newUnboundChanUnit() *unboundChanUnit {
    uc := unboundChanUnit{make(chan struct{}), make(chan struct{})}
    go func(c *unboundChanUnit){c.start()}(&uc)
    return &uc
}

func (uc *unboundChanInt) start(){
    buffer := []int{}
    for{
        for len(buffer) > 0 {
            select {
                case uc.Out <- buffer[0]:
                    buffer = buffer[1:]
                case d := <- uc.In:
                    buffer = append(buffer, d)
            }
        }
        for len(buffer) == 0 {
            d := <- uc.In
            buffer = append(buffer, d)
        }
    }
}
func newUnboundChanInt() *unboundChanInt {
    uc := unboundChanInt{make(chan int), make(chan int)}
    go func(c *unboundChanInt){c.start()}(&uc)
    return &uc
}




type unboundChanString struct {
    In chan string
    Out chan string
}
func (uc *unboundChanString) start(){
    buffer := []string{}
    for{
        for len(buffer) > 0 {
            select {
                case uc.Out <- buffer[0]:
                    buffer = buffer[1:]
                case d := <- uc.In:
                    buffer = append(buffer, d)
            }
        }
        for len(buffer) == 0 {
            d := <- uc.In
            buffer = append(buffer, d)
        }
    }
}
func newUnboundChanString() *unboundChanString {
    uc := unboundChanString{make(chan string), make(chan string)}
    go func(c *unboundChanString){c.start()}(&uc)
    return &uc
}





type unboundChanMessage struct {
    In chan Message
    Out chan Message
}
func (uc *unboundChanMessage) start(){
    buffer := []Message{}
    for{
        for len(buffer) > 0 {
            select {
                case uc.Out <- buffer[0]:
                    buffer = buffer[1:]
                case d := <- uc.In:
                    buffer = append(buffer, d)
            }
        }
        for len(buffer) == 0 {
            d := <- uc.In
            buffer = append(buffer, d)
        }
    }
}
func newUnboundChanMessage() *unboundChanMessage {
    uc := unboundChanMessage{make(chan Message), make(chan Message)}
    go func(c *unboundChanMessage){c.start()}(&uc)
    return &uc
}





type unboundChanConn struct {
    In chan *duplexConn
    Out chan *duplexConn
}
func (uc *unboundChanConn) start(){
    buffer := []*duplexConn{}
    for{
        for len(buffer) > 0 {
            select {
                case uc.Out <- buffer[0]:
                    buffer = buffer[1:]
                case d := <- uc.In:
                    buffer = append(buffer, d)
            }
        }
        for len(buffer) == 0 {
            d := <- uc.In
            buffer = append(buffer, d)
        }
    }
}
func newUnboundChanConn() *unboundChanConn {
    uc := unboundChanConn{make(chan *duplexConn), make(chan *duplexConn)}
    go func(c *unboundChanConn){c.start()}(&uc)
    return &uc
}





type unboundChanMT struct {
    In chan msgTime
    Out chan msgTime
}
func (uc *unboundChanMT) start(){
    buffer := []msgTime{}
    var d msgTime
    stillOpen := true
    for{
        for stillOpen && len(buffer) > 0 {
            select {
                case uc.Out <- buffer[0]:
                    buffer = buffer[1:]
                case d, stillOpen = <- uc.In:
                    if stillOpen {
                        buffer = append(buffer, d)
                    }
            }
        }
        if !stillOpen{
            for _, mt := range buffer {
                uc.Out <- mt
            }
            close(uc.Out)
            return
        } else {
            d, stillOpen := <- uc.In
            if !stillOpen {
                close(uc.Out)
                return
            }
            buffer = append(buffer, d)
        }
    }
}
func newUnboundChanMT() *unboundChanMT {
    uc := unboundChanMT{make(chan msgTime), make(chan msgTime)}
    go func(c *unboundChanMT){c.start()}(&uc)
    return &uc
}
