package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"code.linksmart.eu/hds/historical-datastore/registry"
	"github.com/cisco/senml"
)

func main() {
	hdsURLFlag := flag.String("server", "http://hds:8085", "URL of Historical Datastore service")
	flag.Parse()
	log.Println("Started the Dummy Generator")

	hdsURL := *hdsURLFlag
	log.Println("HDS URL:", hdsURL)

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	postDS := func(name string) registry.DataSource {
		ds := registry.DataSource{
			Resource:  "http://dummy/" + name,
			Retention: "1h",
			Type:      "float",
			Aggregation: []registry.Aggregation{
				registry.Aggregation{
					Interval:   "1m",
					Aggregates: []string{"min", "max"},
					Retention:  "1w",
				},
				registry.Aggregation{
					Interval:   "1h",
					Aggregates: []string{"mean", "stddev"},
					Retention:  "1w",
				},
			},
		}
		jsonValue, _ := json.Marshal(ds)
		for {
			log.Println("Looking for the existing datasource with name", name)
			resp, err := http.Get(hdsURL + "/registry/one/resource/suffix/" + name)
			if err != nil {
				log.Fatalf("Error: %s", err)
			}
			if resp.StatusCode == 200 {
				body, _ := ioutil.ReadAll(resp.Body)
				var reg registry.Registry
				err := json.Unmarshal(body, &reg)
				if err != nil {
					log.Fatalf("Error: %s", err)
				}
				if reg.Total != 0 {
					log.Println("Found datasource", reg.Entries[0].ID)
					break
				}
			} else {
				body, _ := ioutil.ReadAll(resp.Body)
				log.Fatalf("%s: %s", resp.Status, string(body))
			}

			log.Println("Creating datasource named", name)
			resp, err = http.Post(hdsURL+"/registry", "application/json", bytes.NewBuffer(jsonValue))
			if err != nil {
				log.Printf("Error: %s. Retrying...", err)
				time.Sleep(1 * time.Second)
				continue
			}
			if resp.StatusCode == 201 {
				location, err := resp.Location()
				if err != nil {
					log.Fatalln(err)
				}
				ds.ID = strings.Split(location.Path, "/")[2]
				log.Println("Created datasource", ds.ID)
				break
			} else if resp.StatusCode == 409 {
				body, _ := ioutil.ReadAll(resp.Body)
				log.Fatalf("%s: %s. Retrying...", resp.Status, string(body))
			} else {
				body, _ := ioutil.ReadAll(resp.Body)
				log.Printf("%s: %s. Retrying...", resp.Status, string(body))
				time.Sleep(1 * time.Second)
				continue
			}
		}

		return ds
	}

	dss := []registry.DataSource{postDS("ds1"), postDS("ds2")}

	ticker := time.NewTicker(time.Second * 5)
	for range ticker.C {
		for _, ds := range dss {
			rn := r.Float64()
			senmlRecord := senml.SenMLRecord{
				Name:  ds.Resource,
				Value: &rn,
			}
			senmlPack := senml.SenML{Records: []senml.SenMLRecord{senmlRecord}}
			jsonValue, _ := senml.Encode(senmlPack, senml.JSON, senml.OutputOptions{})
			log.Println("Submitting", string(jsonValue))
			resp, err := http.Post(hdsURL+"/data/"+ds.ID, "application/senml+json", bytes.NewBuffer(jsonValue))
			if err != nil {
				log.Printf("Error: %s", err)
			} else if resp.StatusCode != 202 {
				body, _ := ioutil.ReadAll(resp.Body)
				log.Printf("%s: %s", resp.Status, string(body))
			}
		}
	}

}
