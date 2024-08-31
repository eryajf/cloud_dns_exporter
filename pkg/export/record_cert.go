package export

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/eryajf/cloud_dns_exporter/pkg/provider"
)

const (
	maxConcurrency = 100
	timeout        = 10 * time.Second
)

func GetMultipleCertInfo(records []provider.GetRecordCertReq) ([]provider.RecordCert, error) {
	results := make([]provider.RecordCert, len(records))
	semaphore := make(chan struct{}, maxConcurrency)

	var wg sync.WaitGroup
	resultChan := make(chan struct {
		index int
		cert  provider.RecordCert
	}, len(records))
	go func() {
		for result := range resultChan {
			results[result.index] = result.cert
			wg.Done()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for i, record := range records {
		wg.Add(1)
		go func(i int, record provider.GetRecordCertReq) {
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()

				cert, err := GetCertInfo(record)
				if err != nil {
					cert.ErrorMsg = err.Error()
				}
				resultChan <- struct {
					index int
					cert  provider.RecordCert
				}{i, cert}
			case <-ctx.Done():
				resultChan <- struct {
					index int
					cert  provider.RecordCert
				}{i, provider.RecordCert{ErrorMsg: "operation timed out"}}
			}
		}(i, record)
	}
	wg.Wait()
	close(resultChan)

	return results, nil
}

// GetCertInfo 获取证书信息
func GetCertInfo(record provider.GetRecordCertReq) (certInfo provider.RecordCert, err error) {
	config := &tls.Config{
		InsecureSkipVerify: true,
	}
	d := net.Dialer{
		Timeout: time.Second * 3,
	}
	conn, err := tls.DialWithDialer(&d, "tcp", record.FullRecord+":443", config)
	if err != nil {
		return certInfo, err
	}
	defer conn.Close()
	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return certInfo, fmt.Errorf("未找到证书")
	}

	certInfo.CloudProvider = record.CloudProvider
	certInfo.CloudName = record.CloudName
	certInfo.DomainName = record.DomainName
	certInfo.FullRecord = record.FullRecord
	certInfo.RecordID = record.RecordID

	cert := certs[0]
	certInfo.SubjectCommonName = cert.Subject.CommonName
	if strings.Contains(certInfo.SubjectCommonName, record.DomainName) {
		certInfo.CertMatched = true
	} else {
		certInfo.CertMatched = false
		certInfo.ErrorMsg = "证书不匹配"
	}
	if len(cert.Subject.Organization) > 0 {
		certInfo.SubjectOrganization = cert.Subject.Organization[0]
	}
	if len(cert.Subject.OrganizationalUnit) > 0 {
		certInfo.SubjectOrganizationalUnit = cert.Subject.OrganizationalUnit[0]
	}
	// 从证书中提取颁发者信息
	certInfo.IssuerCommonName = cert.Issuer.CommonName
	if len(cert.Issuer.Organization) > 0 {
		certInfo.IssuerOrganization = cert.Issuer.Organization[0]
	}
	if len(cert.Issuer.OrganizationalUnit) > 0 {
		certInfo.IssuerOrganizationalUnit = cert.Issuer.OrganizationalUnit[0]
	}
	// 从证书中提取日期信息
	certInfo.CreatedDate = cert.NotBefore.Format(time.DateOnly)
	certInfo.ExpiryDate = cert.NotAfter.Format(time.DateOnly)
	// 计算距离到期日期还有多少天
	daysUntilExpiry := int(time.Until(cert.NotAfter).Hours() / 24)
	certInfo.DaysUntilExpiry = daysUntilExpiry
	return certInfo, nil
}

// getNewRecord 判断域名解析记录是否符合可获取ssl证书信息的条件
func getNewRecord(records []provider.Record) (newRecords []provider.Record) {
	var wg sync.WaitGroup
	recordChan := make(chan provider.Record)
	for _, record := range records {
		wg.Add(1)
		go func(rec provider.Record) {
			defer wg.Done()
			if rec.RecordName == "@" {
				rec.FullRecord = rec.DomainName
			}
			if strings.Contains(rec.FullRecord, "*") {
				rec.FullRecord = strings.ReplaceAll(rec.FullRecord, "*", "a")
			}
			if (rec.RecordType == "A" || rec.RecordType == "CNAME") &&
				rec.RecordStatus == "enable" && isPortOpen(rec.FullRecord) {
				recordChan <- rec
			}
		}(record)
	}
	go func() {
		wg.Wait()
		close(recordChan)
	}()
	for rec := range recordChan {
		newRecords = append(newRecords, rec)
	}
	return
}

// isPortOpen 检查给定域名的443端口是否通
func isPortOpen(domain string) bool {
	timeout := 1 * time.Second
	conn, err := net.DialTimeout("tcp", domain+":443", timeout)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}
