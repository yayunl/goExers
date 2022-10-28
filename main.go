package main

import (
	"fmt"
	"github.com/cloverstd/tcping/ping"
	"github.com/cloverstd/tcping/ping/http"
	"github.com/cloverstd/tcping/ping/tcp"
	"goExers/apps"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	//"goExers/apps"
	//"log"
	"net"
)

var (
	sigs      chan os.Signal
	dnsServer []string
)

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func init() {
	//ua := rootCmd.Flags().String("user-agent", "tcping", `Use custom UA in http mode.`)

	ping.Register(ping.TCP, func(url *url.URL, op *ping.Option) (ping.Ping, error) {
		port, err := strconv.Atoi(url.Port())
		if err != nil {
			return nil, err
		}
		return tcp.New(url.Hostname(), port, op, false), nil
	})

	ping.Register(ping.HTTP, func(url *url.URL, op *ping.Option) (ping.Ping, error) {

		op.UA = "Use custom UA"
		return http.New("GET", url.String(), op, false)
	})

	ping.Register(ping.HTTPS, func(url *url.URL, op *ping.Option) (ping.Ping, error) {

		op.UA = "Use custom UA"
		return http.New("GET", url.String(), op, false)
	})
}

func tcpHttpPing(ipAddr, pingPort, interval, timeout string, pingAttempts int) {
	// interval: units (s, ms, us)
	// timeout: units (s, ms, us)
	intervalDuration, err := ping.ParseDuration(interval)
	if err != nil {
		fmt.Println("parse interval failed", err)
		//cmd.Usage()
		return
	}

	timeoutDuration, err := ping.ParseDuration(timeout)
	if err != nil {
		fmt.Println("parse timeout failed", err)
		//cmd.Usage()
		return
	}

	// URL stuff
	site := fmt.Sprintf("http://%s", ipAddr)
	url, err := ping.ParseAddress(site)
	if err != nil {
		fmt.Printf("%s is an invalid target.\n", site)
		return
	}

	var defaultPort string
	if pingPort != "" {
		defaultPort = pingPort
	} else {
		defaultPort = "80"
	}

	if port := url.Port(); port != "" {
		defaultPort = port
	} else if url.Scheme == "https" {
		defaultPort = "443"
	}
	port, err := strconv.Atoi(defaultPort)
	if err != nil {
		fmt.Printf("%s is invalid port.\n", defaultPort)
		return
	}

	url.Host = fmt.Sprintf("%s:%d", url.Hostname(), port)

	protocol, err := ping.NewProtocol(url.Scheme)
	if err != nil {
		fmt.Println("invalid protocol", err)
		//cmd.Usage()
		return
	}

	option := ping.Option{
		Timeout: timeoutDuration,
	}

	pingFactory := ping.Load(protocol)
	p, err := pingFactory(url, &option)
	if err != nil {
		fmt.Println("load pinger failed", err)
		//fmt.Usage()
		return
	}

	pinger := ping.NewPinger(os.Stdout, url, p, intervalDuration, pingAttempts)
	sigs = make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go pinger.Ping()
	select {
	case <-sigs:
	case <-pinger.Done():
	}
	pinger.Stop()
	failures, time := pinger.Summarize()
	fmt.Println("Failures: ", failures)
	fmt.Println("time: ", time)
}

func main() {
	maxWorkers := 150

	/* Fast ping */
	args := os.Args[1:]
	ip, ipnet, err := net.ParseCIDR(args[0])
	if err != nil {
		log.Fatal(err)
	}

	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}

	apps.FastPinger(ips, maxWorkers)

	//apps.FastPrinter(maxWorkers)

}
