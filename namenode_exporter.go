package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/log"
)

const (
	namespace = "namenode"
)

var (
	listenAddress  = flag.String("web.listen-address", ":9070", "Address on which to expose metrics and web interface.")
	metricsPath    = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
	namenodeJmxUrl = flag.String("namenode.jmx.url", "http://localhost:50070/jmx", "Hadoop JMX URL.")
)

type Exporter struct {
	url         string
	MissingBlocks prometheus.Gauge
	CapacityTotal prometheus.Gauge
	BlocksTotal prometheus.Gauge
}

func NewExporter(url string) *Exporter {
	return &Exporter{
		url: url,
		MissingBlocks: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "MissingBlocks",
			Help:      "MissingBlocks",
		}),
		CapacityTotal: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "CapacityTotal",
			Help:      "CapacityTotal",
		}),
		BlocksTotal: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "BlocksTotal",
			Help:      "BlocksTotal",
		}),
	}
}

// Describe implements the prometheus.Collector interface.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.MissingBlocks.Describe(ch)
	e.CapacityTotal.Describe(ch)
	e.BlocksTotal.Describe(ch)
}

// Collect implements the prometheus.Collector interface.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	resp, err := http.Get(e.url)
	if err != nil {
		log.Error(err)
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
	}
	var f interface{}
	err = json.Unmarshal(data, &f)
	if err != nil {
		log.Error(err)
	}
	// {"beans":[{"name":"Hadoop:service=NameNode,name=FSNamesystem", ...}, {"name":"java.lang:type=MemoryPool,name=Code Cache", ...}, ...]}
	m := f.(map[string]interface{})
	// [{"name":"Hadoop:service=NameNode,name=FSNamesystem", ...}, {"name":"java.lang:type=MemoryPool,name=Code Cache", ...}, ...]
	var nameList = m["beans"].([]interface{})
	for _, nameData := range nameList {
		nameDataMap := nameData.(map[string]interface{})
		/*
		{
			"name" : "Hadoop:service=NameNode,name=FSNamesystem",
			"modelerType" : "FSNamesystem",
			"tag.Context" : "dfs",
			"tag.HAState" : "active",
			"tag.TotalSyncTimes" : "23 6 ",
			"tag.Hostname" : "CNHORTO7502.line.ism",
			"MissingBlocks" : 0,
			"MissingReplOneBlocks" : 0,
			"ExpiredHeartbeats" : 0,
			"TransactionsSinceLastCheckpoint" : 2007,
			"TransactionsSinceLastLogRoll" : 7,
			"LastWrittenTransactionId" : 172706,
			"LastCheckpointTime" : 1456089173101,
			"CapacityTotal" : 307099828224,
			"CapacityTotalGB" : 286.0,
			"CapacityUsed" : 1471291392,
			"CapacityUsedGB" : 1.0,
			"CapacityRemaining" : 279994568704,
			"CapacityRemainingGB" : 261.0,
			"CapacityUsedNonDFS" : 25633968128,
			"TotalLoad" : 6,
			"SnapshottableDirectories" : 0,
			"Snapshots" : 0,
			"LockQueueLength" : 0,
			"BlocksTotal" : 67,
			"NumFilesUnderConstruction" : 0,
			"NumActiveClients" : 0,
			"FilesTotal" : 184,
			"PendingReplicationBlocks" : 0,
			"UnderReplicatedBlocks" : 0,
			"CorruptBlocks" : 0,
			"ScheduledReplicationBlocks" : 0,
			"PendingDeletionBlocks" : 0,
			"ExcessBlocks" : 0,
			"PostponedMisreplicatedBlocks" : 0,
			"PendingDataNodeMessageCount" : 0,
			"MillisSinceLastLoadedEdits" : 0,
			"BlockCapacity" : 2097152,
			"StaleDataNodes" : 0,
			"TotalFiles" : 184,
			"TotalSyncCount" : 7
		}
		*/
		if nameDataMap["name"] == "Hadoop:service=NameNode,name=FSNamesystem" {
			fmt.Println(nameDataMap)
			e.MissingBlocks.Set(nameDataMap["MissingBlocks"].(float64))
			e.CapacityTotal.Set(nameDataMap["CapacityTotal"].(float64))
			e.BlocksTotal.Set(nameDataMap["BlocksTotal"].(float64))
		}
	}
	e.MissingBlocks.Collect(ch)
	e.CapacityTotal.Collect(ch)
	e.BlocksTotal.Collect(ch)
}

func main() {
	flag.Parse()

	exporter := NewExporter(*namenodeJmxUrl)
	prometheus.MustRegister(exporter)

	log.Printf("Starting Server: %s", *listenAddress)
	http.Handle(*metricsPath, prometheus.Handler())
	err := http.ListenAndServe(*listenAddress, nil)
	if err != nil {
		log.Fatal(err)
	}
}