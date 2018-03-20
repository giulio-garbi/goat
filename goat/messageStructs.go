package goat

type messagePredicate struct {
	message   string
	predicate ClosedPredicate
	invalid   bool
}

type Message struct {
    Id int
    Message Tuple
    Pred ClosedPredicate
}

func makeMessage(messageToSend messagePredicate, mid int) Message{
    if messageToSend.invalid {
		return Message{
			Message:   NewTuple(),
			Pred:      False(),
			Id:        mid,
		}
	} else {
		return Message{
			Message:   decodeTuple(messageToSend.message),
			Pred: messageToSend.predicate,
			Id:        mid,
		}
	}
}
