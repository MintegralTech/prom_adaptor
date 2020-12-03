package model

import (
	"fmt"
	"github.com/prometheus/prometheus/prompb"
	"testing"
)

func TestGetJobName(t *testing.T) {
	l1 := prompb.Label{
		Name:  "name",
		Value: "tim",
	}
	l2 := prompb.Label{
		Name:  "age",
		Value: "12",
	}
	l3 := prompb.Label{
		Name:  "sex",
		Value: "man",
	}
	l4 := prompb.Label{
		Name:  "region",
		Value: "cn",
	}

	sli := make([]*prompb.Label, 4)
	sli[0] = &l1
	sli[1] = &l2
	sli[2] = &l3
	sli[3] = &l4

	sli2 := make([]*prompb.Label, 4)
	copy(sli2, sli)
	for i, _ := range sli {
		fmt.Printf("sli=%p, sli2=%p\n", sli[i], sli2[i])
	}
	sli2[2].Value = "fuck"
	fmt.Println(sli2)
	fmt.Println(sli)

}
