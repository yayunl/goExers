package main

import (
	"goExers/apps"
	"net"
)

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func main() {
	maxWorkers := 150

	/* Fast ping */
	//args := os.Args[1:]
	//ip, ipnet, err := net.ParseCIDR(args[0])
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//var ips []string
	//for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
	//	ips = append(ips, ip.String())
	//}
	//
	//apps.FastPinger(ips, maxWorkers)

	apps.FastPrinter(maxWorkers)
}
