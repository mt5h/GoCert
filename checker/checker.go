package checker

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"time"
)

type response struct {
	Message any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

type certData struct {
	Subject            string      `json:"subject"`
	DnsNames           []string    `json:"dns_names,omitempty"`
	Issuer             string      `json:"issuer"`
	NotBefore          string      `json:"not_before"`
	NotAfter           string      `json:"not_after"`
	SerialNumber       string      `json:"serial_number"`
	IpAddresses        []string    `json:"ip_addresses,omitempty"`
	PublicKeyAlgorithm string      `json:"public_key_algorithm"`
	ValidHost          bool        `json:"is_valid_hostname"`
	Chain              []chainCert `json:"parent_certs"`
}

type chainCert struct {
	IsCA        bool   `json:"is_ca"`
	CommonName  string `json:"common_name"`
	Issuer      string `json:"issuer"`
	Exipiration string `json:"expiration"`
}

func GetJsonCert(endpoint string, timeout time.Duration) string {
	cert, err := checkCert(endpoint, timeout)
	var buffer bytes.Buffer
	var msg response

	if err != nil {
		msg.Error = err.Error()
	} else {
		msg.Message = cert
		msg.Error = ""
	}

	err = prettyEncode(msg, &buffer)
	if err != nil {
		fmt.Println("Can not encode certificate info", err.Error())
	}

	return fmt.Sprintf("%s", &buffer)

}

func prettyEncode(data interface{}, out io.Writer) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "    ")
	if err := enc.Encode(data); err != nil {
		return err
	}
	return nil
}

func checkCert(endpointUrl string, timeout time.Duration) (certData, error) {
	conf := &tls.Config{
		InsecureSkipVerify: true,
	}

	u, err := url.Parse(endpointUrl)

	if err != nil {
		return certData{}, err
	}

	endpointHostname := u.Hostname()
	endpointPort := u.Port()

	if endpointPort == "" {
		if u.Scheme == "https" {
			endpointPort = "443"
		}
	}

	dialerConfig := net.Dialer{
		Timeout: timeout,
	}
	// conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%s", endpointHostname, endpointPort), conf)
	conn, err := tls.DialWithDialer(&dialerConfig, "tcp", fmt.Sprintf("%s:%s", endpointHostname, endpointPort), conf)
	if err != nil {
		return certData{}, err
	}

	defer conn.Close()
	certs := conn.ConnectionState().PeerCertificates

	result := certData{
		Subject:            conn.ConnectionState().PeerCertificates[0].Subject.String(),
		DnsNames:           conn.ConnectionState().PeerCertificates[0].DNSNames,
		Issuer:             conn.ConnectionState().PeerCertificates[0].Issuer.String(),
		NotBefore:          conn.ConnectionState().PeerCertificates[0].NotBefore.Format("2006-January-02"),
		NotAfter:           conn.ConnectionState().PeerCertificates[0].NotAfter.Format("2006-January-02"),
		SerialNumber:       conn.ConnectionState().PeerCertificates[0].SerialNumber.String(),
		PublicKeyAlgorithm: conn.ConnectionState().PeerCertificates[0].PublicKeyAlgorithm.String(),
		ValidHost:          true,
	}

	for _, ip := range conn.ConnectionState().PeerCertificates[0].IPAddresses {
		result.IpAddresses = append(result.IpAddresses, ip.String())
	}

	err = conn.ConnectionState().PeerCertificates[0].VerifyHostname(endpointHostname)
	if err != nil {
		result.ValidHost = false
	}

	for _, cert := range certs {
		parentCert := &chainCert{
			Issuer:      cert.Issuer.String(),
			IsCA:        cert.IsCA,
			Exipiration: cert.NotAfter.Format("2006-January-02"),
			CommonName:  cert.Issuer.CommonName,
		}

		result.Chain = append(result.Chain, *parentCert)
	}

	return result, nil

}
