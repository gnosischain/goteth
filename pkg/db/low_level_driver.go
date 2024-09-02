package db

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/ClickHouse/ch-go"
	"github.com/ClickHouse/ch-go/proto"
)

func (s *DBService) ConnectLowLevel(retry bool) error {
	ctx := context.Background()

	opts, migrationUrl := ParseChUrlIntoOptionsLowLevel(s.connectionUrl)

	lowLevelConn, err := ch.Dial(ctx, opts)
	if err == nil {
		s.lowLevelClient = lowLevelConn
		s.migrationUrl = migrationUrl
		log.Info("low level client is connected")
		// err = s.makeMigrations(migrationUrl)
	} else {
		if retry {
			// database may be in idle state, wait 30 seconds and retry another time
			log.Warning("database could be idle, service will retry to connect another time in 30 seconds ...")
			time.Sleep(30 * time.Second)
			return s.ConnectLowLevel(false)
		}
	}

	return err

}

func ParseChUrlIntoOptionsLowLevel(url string) (ch.Options, string) {
	var user string
	var password string
	var database string

	protocolAndDetails := strings.Split(url, "://")
	// protocol := protocolAndDetails[0]
	details := protocolAndDetails[1]

	credentialsAndEndpoint := strings.Split(details, "@")
	credentials := credentialsAndEndpoint[0]
	endpoint := credentialsAndEndpoint[1]

	hostPortAndPathParams := strings.Split(endpoint, "/")
	fqdn := hostPortAndPathParams[0]
	pathParams := hostPortAndPathParams[1]

	pathAndParams := strings.Split(pathParams, "?")
	database = pathAndParams[0]
	// params := pathAndParams[1]

	user = strings.Split(credentials, ":")[0]
	password = strings.Split(credentials, ":")[1]

	log.Print(fqdn)

	options := ch.Options{
		Address:  fqdn,
		Database: database,
		User:     user,
		Password: password,
	}

	migrationDatabaseUrl := fmt.Sprintf("%s?x-multi-statement=true", url)
	if strings.Contains(fqdn, "clickhouse.cloud") {
		options.TLS = &tls.Config{}
		migrationDatabaseUrl = fmt.Sprintf("%s?x-multi-statement=true&x-migrations-table-engine=MergeTree&secure=true", url)
	}

	return options, migrationDatabaseUrl
}

func (p *DBService) Persist(
	query string,
	table string,
	input proto.Input,
	rows int) error {

	startTime := time.Now()

	p.lowMu.Lock()
	err := p.lowLevelClient.Do(p.ctx, ch.Query{
		Body:  query,
		Input: input,
	})
	p.lowMu.Unlock()
	elapsedTime := time.Since(startTime)

	if err == nil {
		log.Debugf("table %s persisted %d rows in %fs", table, rows, elapsedTime.Seconds())

		p.metricsMu.Lock()
		p.monitorMetrics[table].addNewPersist(rows, elapsedTime)
		p.metricsMu.Unlock()
	}

	return err
}
