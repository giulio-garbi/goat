package goat

type signaling struct {
    chnEvt chan struct{}
    chnSignal chan struct{}
    chnSignaled chan struct{}
    chnGet chan chan struct{}
}

func (s *signaling) goroutine() {
    for{
        select {
            case <-s.chnSignal :
                close(s.chnEvt)
                s.chnEvt = make(chan struct{})
                s.chnSignaled <- struct{}{}
            case s.chnGet <- s.chnEvt:
        }
    }
}

func (s *signaling) Get() chan struct{} {
    return <- s.chnGet
}

func (s *signaling) Signal() {
    s.chnSignal <- struct{}{}
    <- s.chnSignaled
}

func newSignaling() *signaling {
    s := signaling{make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan chan struct{})}
    go func(){s.goroutine()}()
    return &s
}
