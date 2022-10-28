package apps

import (
	"bytes"
	"context"
	"fmt"
	"github.com/tatsushid/go-fastping"
	"goExers/patterns"
	"net"
	"os"
	"sort"
	"time"
)

type response struct {
	ipAddr *net.IPAddr
	active bool
	rtt    time.Duration
}

// Define a custom job

var (
	fastPing = func(ipAddrs []any) patterns.Result {
		// create a new pinger
		p := fastping.NewPinger()

		// Resolve the input ip address
		ra, err := net.ResolveIPAddr("ip4:icmp", fmt.Sprintf("%v", ipAddrs[0]))
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
			*resp = response{ipAddr: addr, rtt: t, active: true}
		}

		err = p.Run()
		p.Stop()

		return patterns.Result{Value: resp, Err: nil, JobID: ipAddrs[0]}

	}
)

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

func sortIPs(ips []string) []net.IP {
	realIPs := []net.IP{}

	for _, ip := range ips {
		realIPs = append(realIPs, net.ParseIP(ip))
	}

	sort.Slice(realIPs, func(i, j int) bool {
		return bytes.Compare(realIPs[i], realIPs[j]) < 0
	})

	//for _, ip := range realIPs {
	//	fmt.Printf("%s\n", ip)
	//}

	return realIPs
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
	activeIPs := []string{}
	for resp := range activeEPs(results) {
		activeIPs = append(activeIPs, fmt.Sprintf("%v", resp.ipAddr))
	}
	sortedIPs := sortIPs(activeIPs)

	// reporting
	fmt.Println("Scanned IPs: ", len(ips))
	fmt.Printf("Oneline IPs: %d (%.2f%%) \n", len(activeIPs), float32(len(activeIPs))/float32(len(ips))*100)

	fmt.Println("Time elapsed: ", time.Since(start))
	fmt.Println("Active IP Addresses:")
	for i, ip := range sortedIPs {
		i += 1
		fmt.Printf("#%d:\t %s\n", i, ip)
	}
}
