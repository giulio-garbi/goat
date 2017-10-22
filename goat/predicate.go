package goat

import (
    "fmt"
)

/*
Predicate represents a predicate to be satisfied by the receiver component of
a message sent.
*/
type ClosedPredicate interface {
    //ImmediateSatisfy() (bool,bool)
    Satisfy(*Attributes) bool
    String() string
}

type Predicate interface {
    CloseUnder(*Attributes) ClosedPredicate
}

/*
Equal represents a predicate that is true iff the receiver component has the
attributes Attr1 and Attr2 both set and they evaluate to the same value.
*/

type ccomp struct {
    Par1 interface{}
    IsAttr1 bool
    Op string
    Par2 interface{}
    IsAttr2 bool
}
func (eq ccomp) Satisfy(attr *Attributes) bool {
    a1Val, a1Exists := toValue(attr, eq.Par1, eq.IsAttr1)
    a2Val, a2Exists := toValue(attr, eq.Par2, eq.IsAttr2)
    //fmt.Println(eq.Par1, eq.IsAttr1, a1Val, a1Exists, eq.Op, eq.Par2, eq.IsAttr2, a2Val, a2Exists)
    if !a1Exists || !a2Exists {
        return false
    }
    switch a1 := a1Val.(type){
        case int:{
            a2, isA2Int := a2Val.(int)
            if isA2Int{
                switch eq.Op {
                    case "==":
                        return a1 == a2
                    case "!=":
                        return a1 != a2
                    case "<":
                        return a1 < a2
                    case "<=":
                        return a1 <= a2
                    case ">":
                        return a1 > a2
                    case ">=":
                        return a1 >= a2
                }
            }
        }
        case string:{
            a2, isA2String := a2Val.(string)
            if isA2String{
                switch eq.Op {
                    case "==":
                        return a1 == a2
                    case "!=":
                        return a1 != a2
                    case "<":
                        return a1 < a2
                    case "<=":
                        return a1 <= a2
                    case ">":
                        return a1 > a2
                    case ">=":
                        return a1 >= a2
                }
            }
        }
        case bool:{
            a2, isA2Bool := a2Val.(bool)
            if isA2Bool{
                switch eq.Op {
                    case "==":
                        return a1 == a2
                    case "!=":
                        return a1 != a2
                }
            }
        }
    }
    return false
}
/*func (eq ccomp) ImmediateSatisfy() (bool, bool) {
    return false, false
}*/
func (eq ccomp) String() string {
    return fmt.Sprintf("%s(%s,%s)", GetOpLetter(eq.Op), escapeWithType(eq.Par1, eq.IsAttr1), escapeWithType(eq.Par2, eq.IsAttr2))
}

type compattr struct {
    name string
}
func Comp(atName string) compattr{
    return compattr{atName}
}

type recattr struct {
    name string
}
func Receiver(atName string) recattr{
    return recattr{atName}
}

type comp struct {
    arg1 interface{}
    op string
    arg2 interface{}
}

func closure(arg interface{}, attr *Attributes) (interface{}, bool){
    switch _arg := arg.(type) {
        case compattr: {
            return attr.GetValue(_arg.name), false
        }
        case recattr: {
            return _arg.name, true
        }
        default: {
            return _arg, false
        }
    }
}

func (cmp comp) CloseUnder(attr *Attributes) ClosedPredicate{
    Par1, IsAttr1 := closure(cmp.arg1, attr)
    Par2, IsAttr2 := closure(cmp.arg2, attr)
    
    return ccomp{Par1, IsAttr1, cmp.op, Par2, IsAttr2}
}

func Equals(arg1 interface{}, arg2 interface{}) comp{
    return comp{arg1, "==", arg2}
}

func NotEquals(arg1 interface{}, arg2 interface{}) comp{
    return comp{arg1, "!=", arg2}
}

func LessThan(arg1 interface{}, arg2 interface{}) comp{
    return comp{arg1, "<", arg2}
}

func LessThanOrEqual(arg1 interface{}, arg2 interface{}) comp{
    return comp{arg1, "<=", arg2}
}

func GreaterThan(arg1 interface{}, arg2 interface{}) comp{
    return comp{arg1, ">", arg2}
}

func GreaterThanOrEqual(arg1 interface{}, arg2 interface{}) comp{
    return comp{arg1, ">=", arg2}
}

func Comparison(Par1 interface{}, IsAttr1 bool, Op string, Par2 interface{}, IsAttr2 bool)  comp{
    var arg1 interface{}
    if IsAttr1 {
        arg1 = Receiver(Par1.(string))
    } else {
        arg1 = Par1
    }
    
    var arg2 interface{}
    if IsAttr2 {
        arg2 = Receiver(Par2.(string))
    } else {
        arg2 = Par2
    }
    
    return comp{arg1, Op, arg2}
}

func GetOpLetter(op string) string{
    switch op {
        case "==":
            return "="
        case "!=":
            return "N"
        case "<", ">":
            return op
        case "<=":
            return "l"
        case ">=":
            return "g"
        default: //ERROR
            return "Q"
    }
}

func GetLetterOp(letter string) string{
    switch letter {
        case "=":
            return "=="
        case "N":
            return "!="
        case "<", ">":
            return letter
        case "l":
            return "<="
        case "g":
            return ">="
        default: //ERROR
            return "@ERR@"
    }
}

/*
And represents a predicate that is true iff both the predicates P1 and P2 are true.
*/
type cand struct {
    p1 ClosedPredicate
    p2 ClosedPredicate
}
func (a cand) Satisfy(attr *Attributes) bool {
    return a.p1.Satisfy(attr) && a.p2.Satisfy(attr)
}
/*func (and _and) ImmediateSatisfy() (bool, bool) {
    eval1, can1 := and.p1.ImmediateSatisfy()
    eval2, can2 := and.p2.ImmediateSatisfy()
    if can1 && can2 {
        return eval1 && eval2, true
    } else if (can1 && !eval1) || (can2 && !eval2){
        return false, true
    } else {
        return false, false
    }
}*/
func (a cand) String() string {
    return fmt.Sprintf("&(%s,%s)", a.p1, a.p2)
}

type and struct {
    p1 Predicate
    p2 Predicate
}

func And(p1 Predicate, p2 Predicate, pn ...Predicate) and {
    andPred := and{p1, p2}
    for _, pi := range pn {
        andPred = and{andPred, pi}
    }
    return andPred
}

func (a and) CloseUnder(attr *Attributes) ClosedPredicate{
    return cand{a.p1.CloseUnder(attr), a.p2.CloseUnder(attr)}
}

/*
Or represents a predicate that is true iff either P1 or P2 is true (or both).
*/
type cor struct {
    p1 ClosedPredicate
    p2 ClosedPredicate
}
func (o cor) Satisfy(attr *Attributes) bool {
    return o.p1.Satisfy(attr) || o.p2.Satisfy(attr)
}
/*func (o cor) ImmediateSatisfy() (bool, bool) {
    eval1, can1 := o.p1.ImmediateSatisfy()
    eval2, can2 := o.p2.ImmediateSatisfy()
    if can1 && can2 {
        return eval1 || eval2, true
    } else if (can1 && eval1) || (can2 && eval2){
        return true, true
    } else {
        return false, false
    }
}*/
func (o cor) String() string {
    return fmt.Sprintf("|(%s,%s)", o.p1, o.p2)
}

type or struct {
    p1 Predicate
    p2 Predicate
}
func Or(p1 Predicate, p2 Predicate, pn ...Predicate) or {
    orPred := or{p1, p2}
    for _, pi := range pn {
        orPred = or{orPred, pi}
    }
    return orPred
}
func (o or) CloseUnder(attr *Attributes) ClosedPredicate {
    return cor{o.p1.CloseUnder(attr), o.p2.CloseUnder(attr)}
}

/*
Not represents a predicate that is true iff the predicate P is false.
*/
type cnot struct {
    p ClosedPredicate
}
func (n cnot) Satisfy(attr *Attributes) bool {
    return !n.p.Satisfy(attr)
}
/*func (n cnot) ImmediateSatisfy() (bool, bool) {
    eval, can := not.p.ImmediateSatisfy()
    if can {
        return !eval, true
    } else {
        return false, false
    }
}*/
func (n cnot) String() string {
    return fmt.Sprintf("!(%s)", n.p)
}

type not struct {
    p Predicate
}

func Not(p Predicate) not {
    return not{p}
}

func (n not) CloseUnder(attr *Attributes) ClosedPredicate {
    return cnot{n.p.CloseUnder(attr)}
}


/*
True represents a predicate that is always true.
*/
type _true struct {}
func (t _true) Satisfy(*Attributes) bool {
    return true
}
func (t _true) ImmediateSatisfy() (bool, bool) {
    return true, true
}
func (t _true) String() string {
    return "TT"
}
func True() _true {
    return _true{}
}
func (t _true) CloseUnder(attr *Attributes) ClosedPredicate {
    return t
}

/*
False represents a predicate that is always false.
*/
type _false struct {}
func (f _false) Satisfy(*Attributes) bool {
    return false
}
func (f _false) ImmediateSatisfy() (bool, bool) {
    return false, true
}
func (f _false) String() string {
    return "FF"
}
func False() _false {
    return _false{}
}
func (f _false) CloseUnder(attr *Attributes) ClosedPredicate {
    return f
}

/////////////

func ToPredicate(s string) (ClosedPredicate, error){
    p, _, err := toPredicateInt(s, 0)
    return p, err
}

func toPredicateInt(s string, from int) (ClosedPredicate, int, error) {
    escapedS := (s)
    switch s[from: from+2] {
        case "=(", "N(", "l(", "<(", "g(", ">(":
            attr1, is1Attr, commaPos := unescapeWithType(escapedS, from+2)
            attr2, is2Attr, bracketPos := unescapeWithType(escapedS, commaPos+1)
            return ccomp{attr1, is1Attr, GetLetterOp(s[from:from+1]), attr2, is2Attr}, bracketPos+1, nil
        case "&(":
            p1, commaPos, _ := toPredicateInt(s, from+2)
            p2, bracketPos, _ := toPredicateInt(s, commaPos+1)
            return cand{p1, p2}, bracketPos+1, nil
        case "|(":
            p1, commaPos, _ := toPredicateInt(s, from+2)
            p2, bracketPos, _ := toPredicateInt(s, commaPos+1)
            return cor{p1, p2}, bracketPos+1, nil
        case "!(":
            p, bracketPos, _ := toPredicateInt(s, from+2)
            return cnot{p}, bracketPos+1, nil
        case "TT":
            return _true{}, from+2, nil
        case "FF":
            return _false{}, from+2, nil
        default:
            //TODO Error!
            return nil, from + 1, nil
    }
}
