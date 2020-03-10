// Copyright 2016 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//package main
//
//import (
//	"fmt"
//	"io/ioutil"
//	"log"
//    "sync/atomic"
//	"net/http"
//
//	"github.com/gogo/protobuf/proto"
//	"github.com/golang/snappy"
//	"github.com/prometheus/common/model"
//
//	"github.com/prometheus/prometheus/prompb"
//)
//
//func main() {
//    var index int64
//	http.HandleFunc("/receive", func(w http.ResponseWriter, r *http.Request) {
//        atomic.AddInt64(&index, 1)
//        idx := atomic.LoadInt64(&index)
//        fmt.Println("idx: ",idx)
//		compressed, err := ioutil.ReadAll(r.Body)
//		if err != nil {
//			http.Error(w, err.Error(), http.StatusInternalServerError)
//			return
//		}
//
//		reqBuf, err := snappy.Decode(nil, compressed)
//		if err != nil {
//			http.Error(w, err.Error(), http.StatusBadRequest)
//			return
//		}
//
//		var req prompb.WriteRequest
//		if err := proto.Unmarshal(reqBuf, &req); err != nil {
//			http.Error(w, err.Error(), http.StatusBadRequest)
//			return
//		}
//		for _, ts := range req.Timeseries {
//			fmt.Println(ts)
//			m := make(model.Metric, len(ts.Labels))
//			for _, l := range ts.Labels {
//				m[model.LabelName(l.Name)] = model.LabelValue(l.Value)
//			}
//			fmt.Print(idx, m)
//
//			for _, s := range ts.Samples {
//				fmt.Printf("  %f %d\n", s.Value, s.Timestamp)
//			}
//		}
//	})
//
//	log.Fatal(http.ListenAndServe(":1234", nil))
//}






