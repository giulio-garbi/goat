package goat

import (
    "fmt"
)

/*
Predicate represents a predicate to be satisfied by the receiver component of
a message sent.
*/
type Predicate interface {
    ImmediateSatisfy() (bool,bool)
    Satisfy(*Attributes) bool
    String() string
}

/*
Equal represents a predicate that is true iff the receiver component has the
attributes Attr1 and Attr2 both set and they evaluate to the same value.
*/

type Comp struct {
    Par1 interface{}
    IsAttr1 bool
    Op string
    Par2 interface{}
    IsAttr2 bool
}
func (eq Comp) Satisfy(attr *Attributes) bool {
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
func (eq Comp) ImmediateSatisfy() (bool, bool) {
    return false, false
}
func (eq Comp) String() string {
    return fmt.Sprintf("%s(%s,%s)", GetOpLetter(eq.Op), escapeWithType(eq.Par1, eq.IsAttr1), escapeWithType(eq.Par2, eq.IsAttr2))
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
type And struct {
    P1 Predicate
    P2 Predicate
}
func (and And) Satisfy(attr *Attributes) bool {
    return and.P1.Satisfy(attr) && and.P2.Satisfy(attr)
}
func (and And) ImmediateSatisfy() (bool, bool) {
    eval1, can1 := and.P1.ImmediateSatisfy()
    eval2, can2 := and.P2.ImmediateSatisfy()
    if can1 && can2 {
        return eval1 && eval2, true
    } else if (can1 && !eval1) || (can2 && !eval2){
        return false, true
    } else {
        return false, false
    }
}
func (and And) String() string {
    return fmt.Sprintf("&(%s,%s)", and.P1, and.P2)
}

/*
Or represents a predicate that is true iff either P1 or P2 is true (or both).
*/
type Or struct {
    P1 Predicate
    P2 Predicate
}
func (or Or) Satisfy(attr *Attributes) bool {
    return or.P1.Satisfy(attr) || or.P2.Satisfy(attr)
}
func (or Or) ImmediateSatisfy() (bool, bool) {
    eval1, can1 := or.P1.ImmediateSatisfy()
    eval2, can2 := or.P2.ImmediateSatisfy()
    if can1 && can2 {
        return eval1 || eval2, true
    } else if (can1 && eval1) || (can2 && eval2){
        return true, true
    } else {
        return false, false
    }
}
func (or Or) String() string {
    return fmt.Sprintf("|(%s,%s)", or.P1, or.P2)
}

/*
Not represents a predicate that is true iff the predicate P is false.
*/
type Not struct {
    P Predicate
}
func (not Not) Satisfy(attr *Attributes) bool {
    return !not.P.Satisfy(attr)
}
func (not Not) ImmediateSatisfy() (bool, bool) {
    eval, can := not.P.ImmediateSatisfy()
    if can {
        return !eval, true
    } else {
        return false, false
    }
}
func (not Not) String() string {
    return fmt.Sprintf("!(%s)", not.P)
}

/*
True represents a predicate that is always true.
*/
type True struct {}
func (t True) Satisfy(*Attributes) bool {
    return true
}
func (t True) ImmediateSatisfy() (bool, bool) {
    return true, true
}
func (t True) String() string {
    return "TT"
}

/*
And represents a predicate that is always false.
*/
type False struct {}
func (f False) Satisfy(*Attributes) bool {
    return false
}
func (f False) ImmediateSatisfy() (bool, bool) {
    return false, true
}
func (f False) String() string {
    return "FF"
}

func ToPredicate(s string) (Predicate, error){
    p, _, err := toPredicateInt(s, 0)
    return p, err
}

func toPredicateInt(s string, from int) (Predicate, int, error) {
    escapedS := (s)
    switch s[from: from+2] {
        case "=(", "N(", "l(", "<(", "g(", ">(":
            attr1, is1Attr, commaPos := unescapeWithType(escapedS, from+2)
            attr2, is2Attr, bracketPos := unescapeWithType(escapedS, commaPos+1)
            return Comp{attr1, is1Attr, GetLetterOp(s[from:from+1]), attr2, is2Attr}, bracketPos+1, nil
        case "&(":
            p1, commaPos, _ := toPredicateInt(s, from+2)
            p2, bracketPos, _ := toPredicateInt(s, commaPos+1)
            return And{p1, p2}, bracketPos+1, nil
        case "|(":
            p1, commaPos, _ := toPredicateInt(s, from+2)
            p2, bracketPos, _ := toPredicateInt(s, commaPos+1)
            return Or{p1, p2}, bracketPos+1, nil
        case "!(":
            p, bracketPos, _ := toPredicateInt(s, from+2)
            return Not{p}, bracketPos+1, nil
        case "TT":
            return True{}, from+2, nil
        case "FF":
            return False{}, from+2, nil
        default:
            //TODO Error!
            return nil, from + 1, nil
    }
}
