package model

import (
	"fmt"
	"testing"
	"time"
)

func TestNewConfig(t *testing.T) {

	//Conf = NewConfig()
	//fmt.Printf("%#v", Conf)
	now := time.Now().Format("2006-01-02 15:04:05")
	fmt.Println("time=", now)

}
