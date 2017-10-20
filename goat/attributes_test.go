package goat

import (
    "testing"
    //"reflect"
)

func getPrebuiltAttrs() *Attributes {
    attr := Attributes{}
    mp := map[string]interface{}{"arg1":"val1"}
    attr.init(mp)
    return &attr
}

func TestInitCopies(t *testing.T){
    attr := Attributes{}
    mp := map[string]interface{}{"arg1":"val1"}
    attr.init(mp)
    mp["arg1"] = "val2"
    if attr.GetValue("arg1") != "val1" {
        t.Fail()
    }
}

func TestGetExistingValue(t *testing.T){
    attr := getPrebuiltAttrs()
    if v, exists := attr.Get("arg1"); v != "val1" || !exists {
        t.Fail()
    }
}

func TestGetNotExistingValue(t *testing.T){
    attr := getPrebuiltAttrs()
    if v, exists := attr.Get("nonex"); v != nil || exists {
        t.Fail()
    }
}

func TestGetValueExistingValue(t *testing.T){
    attr := getPrebuiltAttrs()
    if attr.GetValue("arg1") != "val1" {
        t.Fail()
    }
}
/*
func TestGetValueNotExistingValue(t *testing.T){
    attr := getPrebuiltAttrs()
    if v := attr.GetValue("nonex"); v != nil {
        t.Fail()
    }
}*/

func TestHasExistingValue(t *testing.T){
    attr := getPrebuiltAttrs()
    if !attr.Has("arg1") {
        t.Fail()
    }
}

func TestHasNotExistingValue(t *testing.T){
    attr := getPrebuiltAttrs()
    if attr.Has("nonex") {
        t.Fail()
    }
}

func TestSet(t *testing.T){
    attr := getPrebuiltAttrs()
    attr.Set("arg2", "val2")
    if attr.GetValue("arg2") != "val2" {
        t.Fail()
    }
}

func TestSetOverwrite(t *testing.T){
    attr := getPrebuiltAttrs()
    attr.Set("arg1", "val2")
    if attr.GetValue("arg1") != "val2" {
        t.Fail()
    }
}

func TestSetRollbackOverwrite(t *testing.T){
    attr := getPrebuiltAttrs()
    attr.Set("arg1", "val2")
    attr.rollback()
    if attr.GetValue("arg1") != "val1" {
        t.Fail()
    }
}

func TestSetRollbackNew(t *testing.T){
    attr := getPrebuiltAttrs()
    attr.Set("arg2", "val2")
    attr.rollback()
    if attr.Has("arg2") {
        t.Fail()
    }
}

func TestSetCommit(t *testing.T){
    attr := getPrebuiltAttrs()
    attr.Set("arg1", "val2")
    commRes := attr.commit()
    if !commRes || attr.GetValue("arg1") != "val2" {
        t.Fail()
    }
}

func TestSetCommitNoRollback(t *testing.T){
    attr := getPrebuiltAttrs()
    attr.Set("arg1", "val2")
    attr.commit()
    attr.Set("arg1", "val3")
    if attr.GetValue("arg1") != "val3" {
        t.Fail()
    }
}

func TestSetCommitAndRollback(t *testing.T){
    attr := getPrebuiltAttrs()
    attr.Set("arg1", "val2")
    attr.commit()
    attr.Set("arg1", "val3")
    attr.rollback()
    if attr.GetValue("arg1") != "val2" {
        t.Fail()
    }
}

func TestSetTwoCommit(t *testing.T){
    attr := getPrebuiltAttrs()
    attr.Set("arg1", "val2")
    attr.commit()
    attr.Set("arg1", "val3")
    comm2 := attr.commit()
    if !comm2 || attr.GetValue("arg1") != "val3" {
        t.Fail()
    }
}
