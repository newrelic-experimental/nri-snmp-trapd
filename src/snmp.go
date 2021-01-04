package main

import (
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/soniah/gosnmp"
)

func connect(targetHost string, targetPort int, v3 bool, community string, username string, authProtocolArg string, privProtocolArg string, authPassphraseArg string, privPassphraseArg string) error {
	if v3 {
		msgFlags := gosnmp.AuthPriv
		authProtocol := gosnmp.MD5
		if authProtocolArg == "MD5" {
			authProtocol = gosnmp.MD5
		} else if authProtocolArg == "SHA" {
			authProtocol = gosnmp.SHA
		} else {
			log.Error(fmt.Sprintf("invalid auth_protocol %v. Defalting to MD5", authProtocol))
		}
		privProtocol := gosnmp.AES
		if privProtocolArg == "AES" {
			privProtocol = gosnmp.AES
		} else if privProtocolArg == "DES" {
			privProtocol = gosnmp.DES
		} else {
			log.Error(fmt.Sprintf("invalid priv_protocol %v. Defaulting to AES", privProtocol))
		}
		if (authProtocolArg != "") && (privProtocolArg != "") {
			msgFlags = gosnmp.AuthPriv
		} else if (authProtocolArg != "") && (privProtocolArg == "") {
			msgFlags = gosnmp.AuthNoPriv
		} else if (authProtocolArg == "") && (privProtocolArg == "") {
			msgFlags = gosnmp.NoAuthNoPriv
		}
		theSNMP = &gosnmp.GoSNMP{
			Target:        targetHost,
			Port:          uint16(targetPort),
			Version:       gosnmp.Version3,
			Timeout:       time.Duration(10) * time.Second,
			SecurityModel: gosnmp.UserSecurityModel,
			MsgFlags:      msgFlags,
			SecurityParameters: &gosnmp.UsmSecurityParameters{UserName: username,
				AuthenticationProtocol:   authProtocol,
				AuthenticationPassphrase: authPassphraseArg,
				PrivacyProtocol:          privProtocol,
				PrivacyPassphrase:        privPassphraseArg,
			},
		}
	} else {
		community := strings.TrimSpace(community)
		theSNMP = &gosnmp.GoSNMP{
			Target:    targetHost,
			Port:      uint16(targetPort),
			Version:   gosnmp.Version2c,
			Community: community,
			Timeout:   time.Duration(10 * time.Second), // Timeout better suited to walking
			MaxOids:   8900,
		}
	}

	err := theSNMP.Connect()
	if err != nil {
		log.Error(err.Error())
		return fmt.Errorf("error connecting to target %s: %s", targetHost, err)
	}
	return nil
}

func disconnect() {
	err := theSNMP.Conn.Close()
	if err != nil {
		log.Error("error disconnecting from target ", err)
	}
}
