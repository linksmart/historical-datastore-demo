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
)

// SenML structs from https://github.com/krylovsk/gosenml
// DataSource structs from https://linksmart.eu/redmine/projects/historical-datastore/repository

// Message is the root SenML variable
type Message struct {
	BaseName  string  `json:"bn,omitempty"`
	BaseTime  int64   `json:"bt,omitempty"`
	BaseUnits string  `json:"bu,omitempty"`
	Version   int     `json:"ver"`
	Entries   []Entry `json:"e"`
}

// Entry is a measurement of Parameter Entry
type Entry struct {
	Name         string   `json:"n,omitempty"`
	Units        string   `json:"u,omitempty"`
	Value        *float64 `json:"v,omitempty"`
	StringValue  *string  `json:"sv,omitempty"`
	BooleanValue *bool    `json:"bv,omitempty"`
	Sum          *float64 `json:"s,omitempty"`
	Time         int64    `json:"t,omitempty"`
	UpdateTime   int64    `json:"ut,omitempty"`
}

// Registry describes a registry of registered Data Sources
type Registry struct {
	// URL is the URL of the Registry API
	URL string `json:"url"`
	// Entries is an array of Data Sources
	Entries []DataSource `json:"entries"`
	// Page is the current page in Entries pagination
	Page int `json:"page"`
	// PerPage is the results per page in Entries pagination
	PerPage int `json:"per_page"`
	// Total is the total #of pages in Entries pagination
	Total int `json:"total"`
}

// DataSource describes a single data source such as a sensor
type DataSource struct {
	// ID is a unique ID of the data source
	ID string `json:"id"`
	// URL is the URL of the Data Source in the Registry API
	URL string `json:"url"`
	// Data is the URL to the data of this Data Source Data API
	Data string `json:"data"`
	// Resource is the URL identifying the corresponding
	// LinkSmart Resource (e.g., @id in the Resource Catalog)
	Resource string `json:"resource"`
	// Meta is a hash-map with optional meta-information
	Meta map[string]interface{} `json:"meta"`
	// Retention is the retention duration for data
	Retention string `json:"retention"`
	// Aggregation is an array of configured aggregations
	Aggregation []Aggregation `json:"aggregation"`
	// Type is the values type used in payload
	Type string `json:"type"`
	// Format is the MIME type of the payload
	Format string `json:"format"`
}

// Aggregation describes a data aggregation for a Data Source
type Aggregation struct {
	ID string `json:"id"`
	// Interval is the aggregation interval
	Interval string `json:"interval"`
	// Data is the URL to the data in the Aggregate API
	Data string `json:"data"`
	// Aggregates is an array of aggregates calculated on each interval
	// Valid values: mean, stddev, sum, min, max, median
	Aggregates []string `json:"aggregates"`
	// Retention is the retention duration
	Retention string `json:"retention"`
}

//func post(endpoint string, contentType string, b []byte) *http.Response, err {
//	resp, err := http.Post(endpoint, contentType, bytes.NewBuffer(b))
//	if err != nil {
//		log.Fatalf("Error: %s", err)
//		return nil, err
//	}
//	log.Printf(resp.Status)
//	return resp
//}

func main() {
	hdsURLFlag := flag.String("server", "http://hds:8085", "URL of Historical Datastore service")
	flag.Parse()
	log.Println("Started the Dummy Generator")

	hdsURL := *hdsURLFlag
	log.Println("HDS URL:", hdsURL)

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	postDS := func(name string) DataSource {
		ds := DataSource{
			Resource:  "http://dummy/" + name,
			Retention: "24h",
			Type:      "float",
			Format:    "application/senml+json",
			Aggregation: []Aggregation{
				Aggregation{
					Interval:   "1m",
					Aggregates: []string{"min", "max"},
				},
				Aggregation{
					Interval:   "5m",
					Aggregates: []string{"mean", "stddev"},
				},
			},
		}
		jsonValue, _ := json.Marshal(ds)
		for {
			log.Println("Creating datasource named", name)
			//resp := post(hdsURL+"/registry", "application/json", jsonValue)
			resp, err := http.Post(hdsURL+"/registry", "application/json", bytes.NewBuffer(jsonValue))
			if err != nil {
				log.Printf("Error: %s. Retrying...", err)
				time.Sleep(1 * time.Second)
			} else if resp.StatusCode == 409 {
				log.Println("Looking for the existing datasource")
				resp, err := http.Get(hdsURL + "/registry/one/resource/suffix/" + name)
				if err != nil {
					log.Fatalf("Error: %s", err)
				}
				body, _ := ioutil.ReadAll(resp.Body)
				json.Unmarshal(body, &ds)
				break
			} else if resp.StatusCode == 201 {
				location, err := resp.Location()
				if err != nil {
					log.Fatalln(err)
				}
				ds.ID = strings.Split(location.Path, "/")[2]
				break
			} else {
				body, _ := ioutil.ReadAll(resp.Body)
				log.Printf("%s: %s. Retrying...", resp.Status, string(body))
				time.Sleep(1 * time.Second)
			}
		}
		log.Println("Created datasource", ds.ID)
		return ds
	}

	dss := []DataSource{postDS("ds1"), postDS("ds2")}

	ticker := time.NewTicker(time.Second * 5)
	for range ticker.C {
		for _, ds := range dss {
			rn := r.Float64()
			senmlEntry := Entry{
				Name:  ds.Resource,
				Value: &rn,
			}
			jsonValue, _ := json.Marshal(Message{Entries: []Entry{senmlEntry}})
			log.Println("Submitting", string(jsonValue))
			//post(hdsURL+"/data/"+ds.ID, "application/senml+json", jsonValue)
			_, err := http.Post(hdsURL+"/data/"+ds.ID, "application/json", bytes.NewBuffer(jsonValue))
			if err != nil {
				log.Printf("Error: %s", err)
			}
		}
	}

}
