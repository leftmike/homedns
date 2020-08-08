package main

/*
- run on BeagleBone connected to DSL modem
*/

import (
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/libdns/libdns"
	ipify "github.com/rdegges/go-ipify"
)

var (
	profileFlag = flag.String("profile", "", "AWS profile to use")
	regionFlag  = flag.String("region", "", "AWS region to use")
	ttlFlag     = flag.String("ttl", "5m", "TTL for DNS record")
	verboseFlag = flag.Bool("v", false, "verbose")
)

func parseHost(arg string) (string, string) {
	host := arg
	if !strings.HasSuffix(host, ".") {
		host += "."
	}
	parts := strings.Split(host, ".")
	if len(parts) < 3 {
		return "", ""
	}
	return host, strings.Join(parts[len(parts)-3:], ".")
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Printf("pid: %d\n", os.Getpid())

	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		log.Fatal("at least one host must be specified")
	}

	domains := map[string][]string{}
	for _, arg := range args {
		host, domain := parseHost(arg)
		if domain == "" {
			log.Fatalf("expected a fully qualified domain name: %s\n", arg)
		}
		if *verboseFlag {
			log.Printf("%s -> host: %s domain: %s\n", arg, host, domain)
		}
		domains[domain] = append(domains[domain], host)
	}

	var cfg aws.Config
	if *profileFlag != "" {
		cfg.Credentials = credentials.NewSharedCredentials("", *profileFlag)
	}
	if *regionFlag != "" {
		cfg.Region = aws.String(*regionFlag)
	}

	ttl, err := time.ParseDuration(*ttlFlag)
	if err != nil {
		log.Fatal(err)
	}

	var currentIP string
	for {
		ip, err := ipify.GetIp()
		if err != nil {
			log.Fatal(err)
		}

		if ip != currentIP {
			log.Printf("ip: %s\n", ip)
			currentIP = ip

			sess, err := session.NewSession(&cfg)
			if err != nil {
				log.Fatal(err)
			}
			svc := route53.New(sess)

			for domain, hosts := range domains {
				recs, err := GetRecords(svc, domain)
				if err != nil {
					log.Fatal(err)
				}
				if *verboseFlag {
					log.Printf("records: %s\n", recs)
				}

				var updates []string
				for _, host := range hosts {
					needUpdate := true
					for _, rec := range recs {
						if rec.Type == "A" && rec.Name == host && rec.Value == ip {
							needUpdate = false
							break
						}
					}
					if needUpdate {
						updates = append(updates, host)
					}
				}

				if len(updates) > 0 {
					var recs []libdns.Record
					for _, host := range updates {
						recs = append(recs,
							libdns.Record{
								Type:  "A",
								Name:  host,
								Value: ip,
								TTL:   ttl,
							})
					}

					log.Printf("updating %d record(s)\n", len(updates))

					err = SetRecords(svc, domain, recs)
					if err != nil {
						log.Fatal(err)
					}

					for _, host := range updates {
						log.Printf("set %s to %s for %s\n", host, ip, ttl)
					}
				}
			}
		}

		log.Printf("sleeping for %s\n", ttl)
		time.Sleep(ttl)
	}
}
