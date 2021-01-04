package main

import (
	"io/ioutil"
	"strings"

	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

// collectionParser is a struct to aid the automatic
// parsing of a collection yaml file
type collectionParser struct {
	Collect []struct {
		Device string       `yaml:"device"`
		Traps  []trapParser `yaml:"traps"`
	}
}

// trapParser is a struct to aid the automatic
// parsing of a collection yaml file
type trapParser struct {
	Name      string         `yaml:"name"`
	Type      string         `yaml:"type"`
	TrapOid   string         `yaml:"trap_oid"`
	EventType string         `yaml:"event_type"`
	Metrics   []metricParser `yaml:"metrics"`
}

// metricParser is a struct to aid the automatic
// parsing of a collection yaml file
type metricParser struct {
	Oid        string `yaml:"oid"`
	MetricName string `yaml:"metric_name"`
}

// End of parser defs

// fully parsed and validated collection
type collection struct {
	Device string
	Traps  []*trap
}

// trap is a validated and simplified
// representation of the requested dataset
type trap struct {
	Name               string
	Type               string
	TrapOid            string
	EventType          string
	MetricOidToNameMap map[string]string
	Device             string
}

// metric is a storage struct containing
// the information of a single metric.
type metric struct {
	oid        string
	MetricName string
}

// parseYaml reads a yaml file and parses it into a collectionParser.
// It validates syntax only and not content
func parseYaml(filename string) (*collectionParser, error) {
	// Read the file
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		log.WithFields(log.Fields{"filename": filename, "error": err.Error()}).Error("Failed to open file")
		return nil, err
	}
	// Parse the file
	var c collectionParser
	if err := yaml.Unmarshal(yamlFile, &c); err != nil {
		log.WithFields(log.Fields{"error": err.Error()}).Error("Failed to parse collection")
		return nil, err
	}
	return &c, nil
}

// parseCollection takes a raw collectionParser and returns
// an slice of trap objects containing the validated configuration
func parseCollection(c *collectionParser) ([]*collection, error) {
	var cols []*collection
	var traps []*trap
	for _, dataSet := range c.Collect {
		for _, trapParser := range dataSet.Traps {
			name := strings.TrimSpace(trapParser.Name)
			_type := strings.TrimSpace(trapParser.Type)
			trapOid := strings.TrimSpace(trapParser.TrapOid)
			eventType := strings.TrimSpace(trapParser.EventType)
			metricOidToNameMap := make(map[string]string)

			metricParsers := trapParser.Metrics
			for _, metricParser := range metricParsers {
				metricOid := strings.TrimSpace(metricParser.Oid)
				//force all oids to start with a leading dot indicating abolute oids as required by gosnmp
				if !strings.HasPrefix(metricOid, ".") {
					metricOid = "." + metricOid
				}
				metricOidToNameMap[metricOid] = metricParser.MetricName
			}
			traps = append(traps, &trap{
				Name:               name,
				Type:               _type,
				TrapOid:            trapOid,
				EventType:          eventType,
				MetricOidToNameMap: metricOidToNameMap,
				Device:             dataSet.Device,
			})
		}
		col := collection{Device: dataSet.Device, Traps: traps}
		cols = append(cols, &col)
	}
	return cols, nil
}
