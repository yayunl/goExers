package apps

import (
	"bytes"
	"context"
	"fmt"
	"github.com/cloverstd/tcping/ping"
	"github.com/cloverstd/tcping/ping/http"
	"github.com/cloverstd/tcping/ping/tcp"
	"github.com/tatsushid/go-fastping"
	"goExers/patterns"
	"net"
	"net/url"
	"os"
	//"os/signal"
	"sort"
	"strconv"
	//"syscall"
	"time"
)

type response struct {
	ipAddr   *net.IPAddr
	active   bool
	rtt      time.Duration
	pingType string
}

type String string

func (s String) String() string {
	return string(s)
}
func tcpHttpPing(ipAddr, pingPort, interval, timeout string, pingAttempts int) (success bool, avgTime time.Duration) {
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

	// Port stuff
	var defaultPort string
	var proto string

	switch pingPort {
	case "":
		defaultPort = "80"
		proto = "http"

	case "443":
		proto = "https"

	default:
		proto = "http"
		defaultPort = "80"
	}

	//if pingPort != "" {
	//	defaultPort = pingPort
	//} else {
	//	defaultPort = "80"
	//	proto = "http"
	//}
	// URL stuff

	site := fmt.Sprintf("%s://%s", proto, ipAddr)
	url, err := ping.ParseAddress(site)
	if err != nil {
		fmt.Printf("%s is an invalid target.\n", site)
		return
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
	//sigs = make(chan os.Signal, 1)
	//signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go pinger.Ping()
	select {
	//case <-sigs:
	case <-pinger.Done():
	}
	pinger.Stop()
	failures, avgTime := pinger.Summarize()
	if failures == 0 {
		return true, avgTime
	}
	return false, avgTime
}

const ICMP = "icmp/port 22"
const HTTP = "http/port 80"
const HTTPS = "https/port 443"
const RDP = "RDP/port 3389"

var (
	fastPing = func(ipAddrs []any) patterns.Result {
		// create a new pinger
		p := fastping.NewPinger()

		ipAddrStr := fmt.Sprintf("%v", ipAddrs[0])
		// Resolve the input ip address
		ra, err := net.ResolveIPAddr("ip4:icmp", ipAddrStr)
		if err != nil {
			fmt.Println(err)
			fmt.Println("ERROR!!")
			os.Exit(1)
		}
		resp := &response{ipAddr: ra, rtt: 0, active: false}

		// Create a map to store the results of a given ip addr
		//results := make(map[string]*response)
		//results[ra.String()] = nil
		p.AddIPAddr(ra)

		// Create two channels for received responses and idle respectively
		onRecv, onIdle := make(chan *response), make(chan bool)
		defer close(onRecv)
		defer close(onIdle)

		p.OnRecv = func(addr *net.IPAddr, t time.Duration) {
			//fmt.Println("received")
			*resp = response{ipAddr: addr, rtt: t, active: true, pingType: ICMP}
		}

		err = p.Run()
		p.Stop()

		// Attempt http ping if icmp failed

		if !resp.active {
			// Try http ping
			httpActive, httpAvgTime := tcpHttpPing(ipAddrStr, "80", "1ms", "1s", 1)

			if httpActive {
				resp = &response{ipAddr: ra, rtt: httpAvgTime, active: true, pingType: HTTP}
				return patterns.Result{Value: resp, Err: nil, JobID: ipAddrs[0]}
			} else {
				// Try https ping
				httpsActive, httpsAvgTime := tcpHttpPing(ipAddrStr, "443", "1ms", "1s", 1)
				if httpsActive {
					resp = &response{ipAddr: ra, rtt: httpsAvgTime, active: httpsActive, pingType: HTTPS}
					return patterns.Result{Value: resp, Err: nil, JobID: ipAddrs[0]}
				}
				// Try RDP ping
				rdpActive, rdpAvgTime := tcpHttpPing(ipAddrStr, "3389", "1ms", "1s", 1)
				if rdpActive {
					resp = &response{ipAddr: ra, rtt: rdpAvgTime, active: httpsActive, pingType: RDP}
					return patterns.Result{Value: resp, Err: nil, JobID: ipAddrs[0]}
				}

				// ICMP, HTTP, RDP and HTTP pings fail
				return patterns.Result{Value: resp, Err: nil, JobID: ipAddrs[0]}
			}

		}

		//ICMP ping success
		return patterns.Result{Value: resp, Err: nil, JobID: ipAddrs[0]}

	}
)

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

func creatTasks(ips []string) []patterns.Job {
	var tasks []patterns.Job
	for i, ip := range ips {
		newTask := patterns.Job{
			ID:   i,
			Fn:   fastPing,
			Args: []any{ip},
		}
		tasks = append(tasks, newTask)
	}

	return tasks
}

func sortResponses(ips []*response) []*response {
	//realIPs := []net.IP{}
	//
	//for _, ip := range ips {
	//	realIPs = append(realIPs, net.ParseIP(ip))
	//}

	sort.Slice(ips, func(i, j int) bool {
		return bytes.Compare(net.ParseIP(fmt.Sprintf("%s", ips[i].ipAddr)),
			net.ParseIP(fmt.Sprintf("%s", ips[j].ipAddr))) < 0
	})

	//for _, ip := range realIPs {
	//	fmt.Printf("%s\n", ip)
	//}

	return ips
}

func FastPinger(ips []string, maxWorkers int) {
	start := time.Now()

	// Create channels for global control
	done := make(chan struct{})
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()
	// Step 1: Create a batch of tasks
	tasks := creatTasks(ips)
	// Step 2: Create a pool of workers
	wp := patterns.New(maxWorkers)
	// Step 3: Assign the tasks to the workers
	wp.Assign(patterns.JobGenerator(done, tasks...))
	// Step 4: Let the workers run the assigned tasks
	go wp.Run(ctx)
	// Step 5: Gather the results
	results := wp.Results()
	activeEPs := func(resultStream <-chan patterns.Result) <-chan *response {
		activeResultStream := make(chan *response)
		go func() {
			defer close(activeResultStream)
			for r := range resultStream {
				resp, ok := r.Value.(*response)
				if ok && resp.active {
					//v := fmt.Sprintf("IP: %s, Active: %v, RTT: %v \n", resp.ipAddr, resp.active, resp.rtt)
					//activeIPs = append(activeIPs, resp)
					activeResultStream <- resp
				}
			}
		}()
		return activeResultStream
	}
	// Gather active IPs and sort them
	var activeResponses []*response
	for resp := range activeEPs(results) {
		activeResponses = append(activeResponses, resp)
	}
	sortedActiveResps := sortResponses(activeResponses)

	// reporting
	fmt.Println("Scanned IPs: ", len(ips))
	fmt.Printf("Alive IPs: %d (%.2f%%) \n", len(sortedActiveResps), float32(len(sortedActiveResps))/float32(len(ips))*100)

	fmt.Println("Time elapsed: ", time.Since(start))
	fmt.Println("Alive IP Addresses:")
	for i, ip := range sortedActiveResps {
		i += 1
		fmt.Printf("#%d:\t %s (%s)\n", i, ip.ipAddr, ip.pingType)
	}
}
