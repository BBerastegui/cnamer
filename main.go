package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

func main() {

	// verbose flag - Will print the domain that has the CNAME as the second parameter in a comma separated line
	var verbose bool
	flag.BoolVar(&verbose, "v", false, "Outputs the domains that generated the cname too in format: cname_value,cname")
	flag.Parse()

	servers := []string{
		"8.8.8.8",
		"8.8.4.4",
		"9.9.9.9",
		"1.1.1.1",
		"1.0.0.1",
		"8.8.8.8",
		"8.8.4.4",
	}

	// Try to grab a better list of DNS servers
	resp, err := http.Get("https://raw.githubusercontent.com/BBerastegui/fresh-dns-servers/master/resolvers.csv")
	body, err := ioutil.ReadAll(resp.Body)
	if err == nil {
		servers = strings.Split(string(body), ",")
	}
	defer resp.Body.Close()

	rand.Seed(time.Now().Unix())

	type job struct{ domain, server string }
	jobs := make(chan job)

	var wg sync.WaitGroup
	// Launch 10 goroutines per server
	for i := 0; i < len(servers)*10; i++ {
		wg.Add(1)

		go func() {
			for j := range jobs {

				cname, err := getCNAME(j.domain, j.server)
				if err != nil {
					//fmt.Println(err)
					continue
				}

				// Remove trailing .
				cname = strings.TrimSuffix(cname, ".")

				if verbose {
					fmt.Printf("%s,%s\n", cname, string(j.domain))
				} else {
					fmt.Printf("%s\n", cname)
				}

			}
			wg.Done()
		}()
	}

	sc := bufio.NewScanner(os.Stdin)

	for sc.Scan() {
		target := strings.ToLower(strings.TrimSpace(sc.Text()))
		if target == "" {
			continue
		}
		server := servers[rand.Intn(len(servers))]

		jobs <- job{target, server}
	}
	close(jobs)

	wg.Wait()

}

func resolves(domain string) bool {
	_, err := net.LookupHost(domain)
	return err == nil
}

func getCNAME(domain, server string) (string, error) {
	c := dns.Client{}

	m := dns.Msg{}
	if domain[len(domain)-1:] != "." {
		domain += "."
	}
	m.SetQuestion(domain, dns.TypeCNAME)
	m.RecursionDesired = true

	r, _, err := c.Exchange(&m, server+":53")
	if err != nil {
		return "", err
	}

	if len(r.Answer) == 0 {
		return "", fmt.Errorf("no answers for %s", domain)
	}

	for _, ans := range r.Answer {
		if r, ok := ans.(*dns.CNAME); ok {
			return r.Target, nil
		}
	}
	return "", fmt.Errorf("no cname for %s", domain)

}
