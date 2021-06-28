package main

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/harrykimpel/newrelic-client-go/newrelic"
	log "github.com/sirupsen/logrus"
	"github.com/soniah/gosnmp"
)

func newProcessor(community string, defaultEventType string, defaultSNMPDevice string, client *newrelic.NewRelic) func(packet *gosnmp.SnmpPacket, addr *net.UDPAddr) {
	processTrap := func(packet *gosnmp.SnmpPacket, addr *net.UDPAddr) {
		sourceIP := addr.IP.String()
		if *verboseFlag {
			log.Info(fmt.Sprintf("Trap received from %s\n", sourceIP))
		}
		if packet.Version == gosnmp.Version2c {
			if packet.Community != community {
				log.Error("Invalid community string")
				return
			}
		}

		ms := make(map[string]interface{})
		ms["trapSource"] = sourceIP

		//ipAddress := net.ParseIP(sourceIP)
		lookupAddr, err := net.LookupAddr(sourceIP)
		sourceHost := sourceIP
		if err != nil {
			log.Error("error LookupAddr "+sourceIP, err)
		} else {
			sourceHost = lookupAddr[0]
		}
		log.Info(fmt.Sprintf("Lookup Addr: %s\n", sourceHost))
		ms["trapSourceFullHostname"] = sourceHost
		shortSourceHost := strings.Split(sourceHost, ".")
		ms["trapSourceShortHostname"] = shortSourceHost[0]

		var trapOidDef *trap
		trapOidLookupOk := false

		//first pass over oid variable to extract trap oid
		for _, v := range packet.Variables {
			if v.Name == ".1.3.6.1.6.3.1.1.4.1.0" || v.Name == "snmpTrapOID" {
				switch trapOidValue := v.Value.(type) {
				case string:
					trapOidDef, trapOidLookupOk = trapOidToStructMap[trapOidValue]
					if trapOidLookupOk {
						ms["name"] = trapOidDef.Name
					} else {
						if *verboseFlag {
							log.Warn(fmt.Sprintf("Lookup of trap OID[%s] failed. Consider adding it to traps.yml configuration", trapOidValue))
						}
						ms["name"] = trapOidValue
					}
				default:
					log.Error(fmt.Sprintf("Unable to handle non string trap_oid type [%s=%T]", v.Name, trapOidValue))
					return
				}
			} else {
				if *verboseFlag {
					log.Warn(fmt.Sprintf("v.Name: %s", v.Name))
				}
			}

		}
		if trapOidLookupOk {
			var msEventType string
			msEventType = defaultEventType
			if trapOidDef.EventType != "" {
				msEventType = trapOidDef.EventType
			}
			ms["eventType"] = msEventType
			ms["device"] = trapOidDef.Device
			populateVariables(ms, packet, trapOidDef)
		} else {
			if dropUndeclaredTraps {
				if *verboseFlag {
					log.Warn(fmt.Sprintf("Ignoring trap: %v", packet))
					//b := trapOidDef. .([]byte)
					//t := packet gosnmp.Default.UnmarshalTrap(packet)
					//errMsg := "Ignoring trap: version %s, ContextName %s, Error %s", packet.Version, packet.ContextName, packet.Error.String()
					log.Warn(fmt.Sprintf("Ignoring trap: version %s, ContextName %s, Error %s", packet.Version, packet.ContextName, packet.Error.String()))
				}
				return
			}

			ms["eventType"] = defaultEventType
			ms["device"] = defaultSNMPDevice
		}
		if *verboseFlag {
			log.Info(fmt.Sprintf("Adding event: %v", ms))
		}
		/*err := client.EnqueueEvent(ms)
		if err != nil {
			log.Error("", err)
		}*/
		// Queueu a custom event.
		if err := client.Events.EnqueueEvent(context.Background(), ms); err != nil {
			log.Fatal("error posting custom event:", err)
		}
	}
	return processTrap
}

func populateVariables(ms map[string]interface{}, packet *gosnmp.SnmpPacket, trapDef *trap) {
	for _, v := range packet.Variables {
		if v.Name == "" {
			continue
		}
		var variableName string
		metricOidToNameMap := trapDef.MetricOidToNameMap
		if metricOidToNameMap != nil {
			label, ok := metricOidToNameMap[v.Name]
			if ok {
				variableName = label
			} else {
				variableDef, ok := metricOidToStructMap[v.Name]
				if ok {
					variableName = variableDef.MetricName
				} else {
					variableName = v.Name
				}
			}
		}

		switch v.Type {
		case gosnmp.OctetString:
			b := v.Value.([]byte)
			ms[variableName] = string(b)
		case gosnmp.TimeTicks:
			ms[variableName] = gosnmp.ToBigInt(v.Value)
		case gosnmp.ObjectIdentifier:
			//OIDs are usually ugly to work with, so give users a chance to translate them to descriptive labels
			switch typedVal := v.Value.(type) {
			case string:
				label, ok := metricValueLabelsMap[typedVal]
				if ok {
					ms[variableName] = label
				} else {
					ms[variableName] = v.Value
				}
			default:
				log.Warn(fmt.Sprintf("unable to handle ObjectIdentifier type - %T", typedVal))
				ms[variableName] = v.Value
			}
		case gosnmp.Integer:
			ms[variableName] = gosnmp.ToBigInt(v.Value)
		case gosnmp.Uinteger32:
			ms[variableName] = gosnmp.ToBigInt(v.Value)
		case gosnmp.Gauge32:
			ms[variableName] = gosnmp.ToBigInt(v.Value)
		case gosnmp.Counter32:
			ms[variableName] = gosnmp.ToBigInt(v.Value)
		case gosnmp.Counter64:
			ms[variableName] = gosnmp.ToBigInt(v.Value)
		default:
			log.Error(fmt.Sprintf("%s=%v[type: %v] is of unknown type", v.Name, v.Value, v.Type))
		}
	}
}
