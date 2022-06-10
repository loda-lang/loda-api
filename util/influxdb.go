package util

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

type InfluxDbClient struct {
	host       string
	username   string
	password   string
	httpClient *http.Client
}

func NewInfluxDbClient(host string, username string, password string) *InfluxDbClient {
	log.Printf("Using InfluxDB %s", host)
	return &InfluxDbClient{
		host:       host,
		username:   username,
		password:   password,
		httpClient: &http.Client{},
	}
}

func (c *InfluxDbClient) Write(name string, labels map[string]string, value int) {
	data := name
	for k, v := range labels {
		data = fmt.Sprintf("%s,%s=%s", data, k, v)
	}
	data = fmt.Sprintf("%s value=%v", data, value)
	url := fmt.Sprintf("%s/write?db=loda", c.host)
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(data))
	if err != nil {
		log.Fatal(err)
	}
	req.SetBasicAuth(c.username, c.password)
	res, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Error writing to InfluxDB: %v", err)
	}
	res.Body.Close()
}
