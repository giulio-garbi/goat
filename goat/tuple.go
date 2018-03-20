package goat

import(
    "bytes"
    "encoding/gob"
    "encoding/base64"
    "log"
)

type Tuple struct{
    Elems []interface{}
}

func NewTuple(elems... interface{}) (Tuple){
    return Tuple{elems}
}

func (t Tuple) IsLong(x int) bool{
    return len(t.Elems) == x
}

func (t Tuple) Length() int{
    return len(t.Elems)
}

func (t Tuple) Get(x int) interface{}{
    return t.Elems[x]
}

func (t *Tuple) Set(index int, x interface{}){
    t.Elems[index] = x
}

func (t *Tuple) Append(x interface{}){
    t.Elems = append(t.Elems, x)
}

func (t *Tuple) Pop(){
    t.Elems = t.Elems[:len(t.Elems)-1]
}

func (t Tuple) Contains(x interface{}) bool {
    for i := 0; i<len(t.Elems); i++ {
        if t.Elems[i] == x {
            return true
        }
    }
    return false
}

func (t *Tuple) CloseUnder(attr *Attributes) Tuple{
    el := make([]interface{}, len(t.Elems))
    for i, v := range t.Elems {
        switch castv := v.(type){
            case compattr: {
                el[i] = attr.GetValue(castv.name)
            }
            case evalattr: {
                paramsTpl := NewTuple(castv.params...)
                paramsTplClosed := paramsTpl.CloseUnder(attr)
                params := make([]interface{}, len(castv.params))
                for i := range castv.params {
                    params[i] = paramsTplClosed.Get(i)
                }
                el[i] = castv.fnc(params...)
            }
            default: {
                el[i] = castv
            }
        }
    }
    return Tuple{el}
}

func (t *Tuple) encode() string{
    var network bytes.Buffer
	enc := gob.NewEncoder(&network)
	err := enc.Encode(t)
	if err != nil {
		log.Fatal("Tuple encoding error:", err)
	}
	return base64.StdEncoding.EncodeToString(network.Bytes())
}

func decodeTuple(encoded string) Tuple{
    decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		log.Fatal("Base64 decoding error:", err)
	}
	network := bytes.NewBuffer(decoded)
	dec := gob.NewDecoder(network)
	var t Tuple
	err = dec.Decode(&t)
	if err != nil {
		log.Fatal("Tuple decoding error:", err)
	}
	return t
}
