package goat

type TreeAgent = RingAgent

func NewTreeAgent(registrationAddress string) *TreeAgent{
    return NewRingAgent(registrationAddress)
}
