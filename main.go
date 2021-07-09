package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/miekg/dns"
	"gopkg.in/yaml.v2"
)

type JSONRequest struct {
	Master string `json:"master"`
	Zone   string `json:"zone"`
}

type Response struct {
	Status string `json:"status, omitempty"`
	Error  string `json:"errormessage, omitempty"`
}

type Config struct {
	Listen     string `yaml:"listen"`
	SourceIPv4 string `yaml:"source-ip4"`
	SourceIPv6 string `yaml:"source-ip6"`
}

var config Config

func axfrHandler(w http.ResponseWriter, r *http.Request) {
	var incomingRequest JSONRequest
	resp := Response{Status: "OK"}

	switch r.Method {
	case "POST":
		body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := r.Body.Close(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := json.Unmarshal(body, &incomingRequest); err != nil {
			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			w.WriteHeader(422) // unprocessable entity
			resp.Error = err.Error()
			resp.Status = "Could not unmarshal json"
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				panic(err)
			}
			return
		}

		resp = axfr(incomingRequest.Zone, incomingRequest.Master)
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(200)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			panic(err)
		}
		return

	default:
		http.Error(w, "Not implemented", http.StatusInternalServerError)
	}
}

func axfr(zone, master string) (resp Response) {
	resp.Status = "Error"
	var sourceIP string

	message := new(dns.Msg)
	transfer := new(dns.Transfer)
	message.SetAxfr(zone)

	// check is the given master IP is v4 or v6
	// from the docs: If ip is not an IPv4 address, To4 returns nil.
	if net.ParseIP(master).To4() != nil {
		sourceIP = config.SourceIPv4
	} else {
		sourceIP = config.SourceIPv6
		master = "[" + master + "]"
	}

	m := master + ":53"
	d := net.Dialer{Timeout: time.Second,
		LocalAddr: &net.TCPAddr{
			IP:   net.ParseIP(sourceIP),
			Port: 0,
		}}
	con, err := d.Dial("tcp", m)
	if err != nil {
		log.Printf("%s", err)
		resp.Error = err.Error()
		return resp
	}
	dnscon := &dns.Conn{Conn: con}
	transfer = &dns.Transfer{Conn: dnscon}
	channel, err := transfer.In(message, master)
	if err != nil {
		log.Printf("%s", err)
		resp.Error = err.Error()
	}

	for r := range channel {
		fmt.Printf("%#v\n", r.Error)
		if r.Error != nil {
			resp.Error = r.Error.Error()
			return resp
		}
	}

	resp.Status = "OK"
	return resp
}

func main() {
	raw, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("could not read config: %s", err)
	}

	config = Config{
		Listen: "127.0.0.1:8080",
	}

	if err := yaml.Unmarshal(raw, &config); err != nil {
		log.Fatalf("Could not parse config: %s", err)
	}

	http.HandleFunc("/axfr", axfrHandler)
	log.Fatal(http.ListenAndServe(config.Listen, nil))
}
