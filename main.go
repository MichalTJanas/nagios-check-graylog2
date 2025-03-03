package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// nagios exit codes
const (
	OK = iota
	WARNING
	CRITICAL
	UNKNOWN
)

// export NCG2=debug
const DEBUG = "NCG2"

// license information
const (
	author       = "Antonino Catinello"
	license      = "BSD"
	year         = "2016 - 2018"
	copyright    = "\u00A9"
	contributers = "kahluagenie, theherodied"
	modified     = "mruediger"
)

var (
	// command line arguments
	link    *string
	user    *string
	pass    *string
	version *bool
	// env debugging variable
	debug string
	// performence data
	pdata string
	//console data
	cdata string
	// version value
	id                    string
	indexwarn             *string
	indexcrit             *string
	uncommitWarn          *string
	uncommitCrit          *string
	processBufferTimeCrit *string
	inputBufferCrit       *string
	critical_node         string
	running_node          string
)

// handle performence data output
func perf(elapsed, total, inputs, tput, index float64, uncommited float64, processBufferTime float64, inputBuffer float64) {
	pdata = fmt.Sprintf("time=%f;;;; total=%.f;;;; sources=%.f;;;; throughput=%.f;;;; index_failures=%.f;;;; uncommited=%.f;;;;; processbuffertime=%.10f;;;;; inputbufferate_m15=%.6f", elapsed, total, inputs, tput, index, uncommited, processBufferTime, inputBuffer)
}
func consoleOutput(elapsed, total_cluster_nodes float64, running_node string, total float64, index float64, tput float64, inputs float64, uncommited float64, processBufferTime float64, inputBuffer float64) string {
	cdata = fmt.Sprintf("All nodes in the Cluster: %v\nRunning nodes:\n%v\n%.f total events processed\n%.f index failures\n%.f throughput\n%.f sources\n%.f uncommited\n%.10f processbuffertime\n%.6f inputbufferrate m_15 \nCheck took %v\n", total_cluster_nodes, running_node, total, index, tput, inputs, uncommited, processBufferTime, inputBuffer, elapsed)
	return cdata
}

// handle args
func init() {
	link = flag.String("l", "http://localhost:12900", "Graylog API URL")
	user = flag.String("u", "", "API username")
	pass = flag.String("p", "", "API password")
	version = flag.Bool("version", false, "Display version and license information. (info)")
	debug = os.Getenv(DEBUG)
	perf(0, 0, 0, 0, 0, 0, 0, 0)
	indexwarn = flag.String("w", "", "Index error warning limit. (optional)")
	indexcrit = flag.String("c", "", "Index error critical limit. (optional)")
	uncommitCrit = flag.String("uc", "", "Uncommited journal entries critical threshold. (optional)")
	processBufferTimeCrit = flag.String("pbtc", "", "Process buffer Time critical threshold in s. (optional)")
	inputBufferCrit = flag.String("ibc", "", "Input buffer rate below critical threshold in events/second. (optional)")
}

// return nagios codes on quit
func quit(status int, message string, err error) {
	var ev string

	switch status {
	case OK:
		ev = "OK"
	case WARNING:
		ev = "WARNING"
	case CRITICAL:
		ev = "CRITICAL"
	case UNKNOWN:
		ev = "UNKNOWN"
	}

	// if debugging is enabled
	// print errors
	if len(debug) != 0 {
		fmt.Println(err)
	}

	fmt.Printf("%s - %s|%s\n", ev, message, pdata)
	os.Exit(status)
}

// parse link
func parse(link *string) string {
	l, err := url.Parse(*link)
	if err != nil {
		quit(UNKNOWN, "Cannot parse given URL.", err)
	}

	if !strings.Contains(l.Host, ":") {
		quit(UNKNOWN, "Port number is missing. Please try "+l.Scheme+"://hostname:port", err)
	}

	if !strings.HasPrefix(l.Scheme, "HTTP") && !strings.HasPrefix(l.Scheme, "http") {
		quit(UNKNOWN, "Only HTTP is supported as protocol.", err)
	}

	return l.Scheme + "://" + l.Host + l.Path
}

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("Version: %v License: %v %v %v %v\nContributers: %v\n", id, license, copyright, year, author, contributers)
		os.Exit(3)
	}

	if len(*user) == 0 || len(*pass) == 0 {
		fmt.Println("API Username/Password is mandatory.")
		flag.PrintDefaults()
		os.Exit(3)
	}

	c := parse(link)
	start := time.Now()

	cluster_nodes := query(c+"/system/cluster/nodes", *user, *pass)
	total_cluster_nodes := cluster_nodes["total"]
	all_cluster_nodes := cluster_nodes["nodes"].([]interface{})

	var run_node_ok, run_node_err = 0, 0

	for _, result := range all_cluster_nodes {

		node := result.(map[string]interface{})
		node_id := node["node_id"]
		node_hostname := node["hostname"]

		run_node := query(c+"/cluster", *user, *pass)
		run_node_lc := run_node[node_id.(string)]

		if run_node_lc == nil {
			critical_node += fmt.Sprintf("Node: %v - not alive", node_hostname.(string))
			run_node_err += 1
		}

		running_node += fmt.Sprintf("\tNode: %v - is alive\n", node_hostname.(string))
		run_node_ok += 1
	}

	if len(critical_node) > 0 {
		quit(CRITICAL, critical_node, nil)
	}

	system := query(c+"/system", *user, *pass)
	if system["is_processing"].(bool) != true {
		quit(CRITICAL, "Service is not processing!", nil)
	}
	if strings.Compare(system["lifecycle"].(string), "running") != 0 {
		quit(WARNING, fmt.Sprintf("lifecycle: %v", system["lifecycle"].(string)), nil)
	}
	if strings.Compare(system["lb_status"].(string), "alive") != 0 {
		quit(WARNING, fmt.Sprintf("lb_status: %v", system["lb_status"].(string)), nil)
	}

	index := query(c+"/system/indexer/failures?limit=1&offset=0", *user, *pass)
	tput := query(c+"/system/throughput", *user, *pass)
	inputs := query(c+"/system/inputs", *user, *pass)

	totalcounts := query(c+"/system/indexer/overview", *user, *pass)
	total := totalcounts["counts"].(map[string]interface{})

	uncommited := query(c+"/system/metrics/org.graylog2.journal.entries-uncommitted", *user, *pass)
	processBufferTime := query(c+"/system/metrics/org.graylog2.shared.buffers.processors.ProcessBufferProcessor.processTime", *user, *pass)
	inputBuffer := query(c+"/system/metrics/org.graylog2.shared.buffers.InputBufferImpl.incomingMessages", *user, *pass)

	elapsed := time.Since(start)

	// generate performance data output
	perf(elapsed.Seconds(), total["events"].(float64), inputs["total"].(float64), tput["throughput"].(float64), index["total"].(float64), uncommited["value"].(float64), processBufferTime["p95"].(float64), inputBuffer["m15_rate"].(float64))

	if len(*indexwarn) == 0 && len(*indexcrit) == 0 && len(*uncommitCrit) == 0 && len(*processBufferTimeCrit) == 0 && len(*inputBufferCrit) == 0 {
		quit(CRITICAL, "no thresholds set", nil)
	}

	if len(*indexwarn) != 0 {
		// convert indexwarn and indexcrit strings to float64 variables for comparison below
		indexwarn2, err := strconv.ParseFloat((*indexwarn), 64)
		indexcrit2, err := strconv.ParseFloat((*indexcrit), 64)
		if err != nil {
			quit(UNKNOWN, "Cannot parse given index warning error value.", err)
		}
		if index["total"].(float64) >= indexwarn2 && index["total"].(float64) < indexcrit2 {
			quit(WARNING, fmt.Sprintf("Index Failure above Warning Limit!\nService is running\n%.f total events processed\n%.f index failures\n%.f throughput\n%.f sources\nCheck took %v\n",
				total["events"].(float64), index["total"].(float64), tput["throughput"].(float64), inputs["total"].(float64), elapsed), nil)
		}
	}
	if len(*indexcrit) != 0 {
		indexcrit2, err := strconv.ParseFloat((*indexcrit), 64)
		if err != nil {
			quit(UNKNOWN, "Cannot parse given index critical error value.", err)
		}
		if index["total"].(float64) >= indexcrit2 {
			quit(CRITICAL, fmt.Sprintf("Index Failure above Critical Limit!\nService is running\n%.f total events processed\n%.f index failures\n%.f throughput\n%.f sources\nCheck took %v\n",
				total["events"].(float64), index["total"].(float64), tput["throughput"].(float64), inputs["total"].(float64), elapsed), nil)
		}
	}
	if len(*uncommitCrit) != 0 {
		uncommitCrit2, err := strconv.ParseFloat((*uncommitCrit), 64)
		if err != nil {
			quit(UNKNOWN, "Cannot parse given uncommited warning error value.", err)
		}
		if uncommited["value"].(float64) > uncommitCrit2 {
			quit(CRITICAL, "Uncommited above Warning Limit!\nService is running\n"+consoleOutput(elapsed.Seconds(), total_cluster_nodes.(float64), running_node, total["events"].(float64), index["total"].(float64), tput["throughput"].(float64), inputs["total"].(float64), uncommited["value"].(float64), processBufferTime["p95"].(float64), inputBuffer["m15_rate"].(float64)), nil)
		}
	}
	if len(*processBufferTimeCrit) != 0 {
		processBufferTimeCrit2, err := strconv.ParseFloat((*processBufferTimeCrit), 64)
		if err != nil {
			quit(UNKNOWN, "Cannot parse given process buffer time critical value.", err)
		}
		if processBufferTime["p95"].(float64) > processBufferTimeCrit2 {
			quit(CRITICAL, "Process Buffer Time critical!\nService is running\n"+consoleOutput(elapsed.Seconds(), total_cluster_nodes.(float64), running_node, total["events"].(float64), index["total"].(float64), tput["throughput"].(float64), inputs["total"].(float64), uncommited["value"].(float64), processBufferTime["p95"].(float64), inputBuffer["m15_rate"].(float64)), nil)
		}
	}
	if len(*inputBufferCrit) != 0 {
		inputBufferCrit2, err := strconv.ParseFloat((*inputBufferCrit), 64)
		if err != nil {
			quit(UNKNOWN, "Cannot parse given input buffer critical value.", err)
		}
		if inputBuffer["m15_rate"].(float64) < inputBufferCrit2 {
			quit(CRITICAL, "Input Buffer rate below threshold!\nService is running\n"+consoleOutput(elapsed.Seconds(), total_cluster_nodes.(float64), running_node, total["events"].(float64), index["total"].(float64), tput["throughput"].(float64), inputs["total"].(float64), uncommited["value"].(float64), processBufferTime["p95"].(float64), inputBuffer["m15_rate"].(float64)), nil)
		}
	}
	quit(OK, "Service is running!\n"+consoleOutput(elapsed.Seconds(), total_cluster_nodes.(float64), running_node, total["events"].(float64), index["total"].(float64), tput["throughput"].(float64), inputs["total"].(float64), uncommited["value"].(float64), processBufferTime["p95"].(float64), inputBuffer["m15_rate"].(float64)), nil)

}

// call Graylog HTTP API
func query(target string, user string, pass string) map[string]interface{} {
	var client *http.Client
	var data map[string]interface{}
	client = &http.Client{}

	req, err := http.NewRequest("GET", target, nil)
	req.SetBasicAuth(user, pass)
	req.Header.Set("Accept", "application/json")

	res, err := client.Do(req)
	if err != nil {
		quit(CRITICAL, "Cannot connect to Graylog API", err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		quit(CRITICAL, "No response received from Graylog API", err)
	}

	if len(debug) != 0 {
		fmt.Println(string(body))
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		quit(UNKNOWN, "Cannot parse JSON from Graylog API", err)
	}

	if res.StatusCode != 200 {
		quit(CRITICAL, fmt.Sprintf("Graylog API replied with HTTP code %v", res.StatusCode), err)
	}

	return data
}
