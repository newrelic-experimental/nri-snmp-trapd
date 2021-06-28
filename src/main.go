package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/harrykimpel/newrelic-client-go/newrelic"
	log "github.com/sirupsen/logrus"
	"github.com/soniah/gosnmp"
	"gopkg.in/yaml.v2"
)

var configFileName *string
var verboseFlag *bool
var dropUndeclaredTraps bool

var theSNMP *gosnmp.GoSNMP

var metricOidToStructMap map[string]*metric
var metricValueLabelsMap map[string]string
var trapOidToStructMap map[string]*trap

//Configuration is
type Configuration struct {
	Collect struct {
		AccountID           string `yaml:"account_id"`
		InsertKey           string `yaml:"insert_key"`
		NRRegion            string `yaml:"nr_region"`
		EventType           string `yaml:"event_type"`
		SNMPDevice          string `yaml:"snmp_device"`
		DropUnDeclaredTraps bool   `yaml:"drop_undeclared_traps"`
		SNMPHost            string `yaml:"snmp_host"`
		SNMPPort            int    `yaml:"snmp_port"`
		HTTPProxyHost       string `yaml:"http_proxy_host"`
		HTTPProxyPort       int    `yaml:"http_proxy_port"`
		Community           string `yaml:"community"`
		V3                  bool   `yaml:"v3"`
		Username            string `yaml:"username"`
		AuthProtocol        string `yaml:"auth_protocol"`
		AuthPassphrase      string `yaml:"auth_passphrase"`
		PrivProtocol        string `yaml:"priv_protocol"`
		PrivPassphrase      string `yaml:"priv_passphrase"`
		TrapDefinitionFiles string `yaml:"trap_definition_files"`

		Params map[string]interface{} `yaml:"params"`
	} `yaml:"collect"`
}

func parseFlags() {
	configFileName = flag.String("config_file", "config.yml", "location of config.yml configuration file")
	verboseFlag = flag.Bool("verbose", false, "verbose logging")
	flag.Parse()
}

func setupLogging() {
	log.SetLevel(log.InfoLevel)
	Formatter := new(log.TextFormatter)
	Formatter.TimestampFormat = "02-01-2006 15:04:05"
	Formatter.FullTimestamp = true
	log.SetFormatter(Formatter)
}

func main() {
	parseFlags()
	setupLogging()

	var config Configuration
	source, err := ioutil.ReadFile(*configFileName)
	if err != nil {
		log.Error(err)
		return
	}
	err = yaml.Unmarshal(source, &config)
	if err != nil {
		log.Error(err)
		return
	}

	insightAccountID := config.Collect.AccountID
	insightInsertKey := config.Collect.InsertKey
	nrRegion := config.Collect.NRRegion
	defaultEventType := config.Collect.EventType
	defaultSNMPDevice := config.Collect.SNMPDevice
	dropUndeclaredTraps = config.Collect.DropUnDeclaredTraps
	snmpHost := config.Collect.SNMPHost
	snmpPort := config.Collect.SNMPPort
	httpProxyHost := config.Collect.HTTPProxyHost
	httpProxyPort := config.Collect.HTTPProxyPort
	community := config.Collect.Community
	v3 := config.Collect.V3
	username := config.Collect.Username
	authProtocol := config.Collect.AuthProtocol
	authPassphrase := config.Collect.AuthPassphrase
	privProtocol := config.Collect.PrivProtocol
	privPassphrase := config.Collect.PrivPassphrase
	collectionFileNames := config.Collect.TrapDefinitionFiles

	if httpProxyHost != "" {
		if httpProxyPort == 0 {
			httpProxyPort = 80
		}
		httpProxy := fmt.Sprintf("http://%s:%d", httpProxyHost, httpProxyPort)
		err = os.Setenv("HTTP_PROXY", httpProxy)
		if err != nil {
			log.Error("Failed to set HTTP_PROXY env variable")
		} else {
			log.Info("Proxy set to " + httpProxy)
		}
	} else {
		log.Info("Proxy not specified")
	}

	//Initalize the metric OID to metric name map
	metricOidToStructMap = make(map[string]*metric)
	//Populate with standard/well known oids (errors etc)
	metricOidToStructMap[".1.3.6.1.2.1.1.3.0"] = &metric{MetricName: "sysUptime"}
	metricOidToStructMap[".1.3.6.1.6.3.1.1.4.1.0"] = &metric{MetricName: "snmpTrapOID"}

	//Initialize the map for translating metric value (when value is OID) to a descriptive name
	metricValueLabelsMap = make(map[string]string)

	//Initialize the snmpTrapOID value labels map
	trapOidToStructMap = make(map[string]*trap)
	trapOidToStructMap[".1.3.6.1.6.3.1.1.5.1"] = &trap{Name: "ColdStart", Type: "trap", TrapOid: ".1.3.6.1.6.3.1.1.5.1", EventType: defaultEventType, Device: defaultSNMPDevice}             // "ColdStart"
	trapOidToStructMap[".1.3.6.1.6.3.1.1.5.2"] = &trap{Name: "WarmStart", Type: "trap", TrapOid: ".1.3.6.1.6.3.1.1.5.2", EventType: defaultEventType, Device: defaultSNMPDevice}             // "WarmStart"
	trapOidToStructMap[".1.3.6.1.6.3.1.1.5.3"] = &trap{Name: "LinkDown", Type: "trap", TrapOid: ".1.3.6.1.6.3.1.1.5.3", EventType: defaultEventType, Device: defaultSNMPDevice}              // "LinkDown"
	trapOidToStructMap[".1.3.6.1.6.3.1.1.5.4"] = &trap{Name: "LinkUp", Type: "trap", TrapOid: ".1.3.6.1.6.3.1.1.5.4", EventType: defaultEventType, Device: defaultSNMPDevice}                // "LinkUp"
	trapOidToStructMap[".1.3.6.1.6.3.1.1.5.5"] = &trap{Name: "AuthenticationFailure", Type: "trap", TrapOid: ".1.3.6.1.6.3.1.1.5.5", EventType: defaultEventType, Device: defaultSNMPDevice} // "AuthenticationFailure"
	trapOidToStructMap[".1.3.6.1.6.3.1.1.5.6"] = &trap{Name: "EGPNeighbourLoss", Type: "trap", TrapOid: ".1.3.6.1.6.3.1.1.5.6", EventType: defaultEventType, Device: defaultSNMPDevice}      // "EGPNeighbourLoss"

	// Ensure a collection file is specified
	if collectionFileNames == "" {
		log.Warn("Trap Configuration files not specified")
	} else {
		// For each collection definition file, parse and collect it
		collectionFiles := strings.Split(collectionFileNames, ",")
		for _, collectionFile := range collectionFiles {
			// Parse the yaml file into a raw definition
			collectionParser, err := parseYaml(collectionFile)
			if err != nil {
				log.Error("failed to parse collection definition file "+collectionFile, err)
				os.Exit(1)
			}
			collections, err := parseCollection(collectionParser)
			if err != nil {
				log.Error("failed to parse collection definition "+collectionFile, err)
				os.Exit(1)
			}

			for _, collection := range collections {
				if err := processCollection(collection); err != nil {
					log.Error("failed to complete collection execution: ", err)
				}
			}
		}
	}

	// Initialize the insights client
	//client := insights.NewInsertClient(insightInsertKey, insightAccountID)
	client, err := newrelic.New(
		newrelic.ConfigInsightsInsertKey(insightInsertKey),
		newrelic.ConfigRegion(nrRegion))
	log.Info("NR insert key is " + insightInsertKey + ", region " + nrRegion)
	if err != nil {
		log.Fatal("error initializing client:", err)
	}
	accountID, err := strconv.Atoi(insightAccountID)
	if err != nil {
		log.Fatal("environment variable NEW_RELIC_ACCOUNT_ID required")
	}
	//cfg := config.New()
	//cfg.InsightsInsertKey = os.Getenv("NEW_RELIC_INSIGHTS_INSERT_KEY")

	/*if validationErr := client.Validate(); validationErr != nil {
		log.Error("error validating NewRelic connection properties", validationErr)
		return
	}*/
	/*if startError := client.Start(); startError != nil {
		log.Error("error creating NewRelic connection", startError)
		return
	}*/
	// Post a custom event.
	if err := client.Events.BatchMode(context.Background(), accountID); err != nil {
		log.Fatal("error starting batch mode:", err)
	}
	//TODO: defer client close

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	exitChannel := make(chan os.Signal, 1)
	signal.Notify(exitChannel, os.Interrupt)

	ticker := time.NewTicker(time.Millisecond * 30000)

	trapEventsChannel := make(chan map[string]interface{}, 500)

	err = connect(snmpHost, snmpPort, v3, community, username, authProtocol, privProtocol, authPassphrase, privPassphrase)
	if err != nil {
		log.Error("error connecting to snmp server "+snmpHost+":"+strconv.Itoa(snmpPort), err)
		os.Exit(1)
	}
	defer disconnect()

	listener := gosnmp.NewTrapListener()
	listener.OnNewTrap = newProcessor(community, defaultEventType, defaultSNMPDevice, client)
	listener.Params = theSNMP

	log.Info("trapd listen address is " + snmpHost + ":" + strconv.Itoa(snmpPort))
	go func() {
		err = listener.Listen(snmpHost + ":" + strconv.Itoa(snmpPort))
		if err != nil {
			log.Error("error starting listener "+snmpHost+":"+strconv.Itoa(snmpPort), err)
			os.Exit(1)
		}
	}()

mainloop:
	for {
		select {
		case <-exitChannel:
			break mainloop
		case <-ctx.Done():
			break mainloop
		case <-ticker.C:
			err = run(ctx, client, trapEventsChannel)
			if err != nil {
				log.Error(err)
			}
		}
	}
	cancel()
	close(exitChannel)
}

func run(ctx context.Context, nrClient *newrelic.NewRelic, trapEventsChannel <-chan map[string]interface{}) error {
	var err error
	err = nrClient.Events.Flush()
	if err != nil {
		return err
	}
	return nil
}

func processCollection(collection *collection) error {
	for _, trap := range collection.Traps {
		trapOid := trap.TrapOid
		if trapOid != "" {
			trapOidToStructMap[trapOid] = trap
		}
		/*
			for _, attribute := range trap.Metrics {
				oid := strings.TrimSpace(attribute.oid)
				metricNameLabelsMap[oid] = attribute
				//All scalar OIDs must end with a .0 suffix by convention.
				//But they are not always specified with their .0 suffix in MIBs and elsewhere
				//So be nice and treat an OID and and its variant with .0 suffix as equivalent
				if !strings.HasSuffix(oid, ".0") {
					metricNameLabelsMap[oid+".0"] = attribute
				}
			}
		*/
	}

	return nil
}
