package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/influxdata/influxdb-client-go/v2"
	"github.com/kefi550/go-healthplanet"
)

var (
	loginId       = os.Getenv("HEALTHPLANET_LOGIN_ID")
	loginPassword = os.Getenv("HEALTHPLANET_LOGIN_PASSWORD")
	clientId      = os.Getenv("HEALTHPLANET_CLIENT_ID")
	clientSecret  = os.Getenv("HEALTHPLANET_CLIENT_SECRET")

	influxdbUrl      = os.Getenv("INFLUXDB_URL")
	influxdbToken    = os.Getenv("INFLUXDB_TOKEN")
	influxdbOrg      = os.Getenv("INFLUXDB_ORG")
	influxdbBucket   = os.Getenv("INFLUXDB_BUCKET")
	influxdbMeasurement = os.Getenv("INFLUXDB_MEASUREMENT")
)

func WriteInfluxDB(host, token, org, bucket, measurement string, tag string, value float64, t time.Time) error {
	client := influxdb2.NewClient(host, token)
	writeAPI := client.WriteAPIBlocking(org, bucket)

	fmt.Printf("Writing to InfluxDB: tag=%s, value=%f, time=%s\n", tag, value, t.Format(time.RFC3339))
	p := influxdb2.NewPointWithMeasurement(measurement).
		AddTag("tag", tag).
		AddField("field", value).
		SetTime(t)
	err := writeAPI.WritePoint(context.Background(), p)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	hp := healthplanet.NewClient(
		loginId,
		loginPassword,
		clientId,
		clientSecret,
	)

	jst, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		log.Fatal(err)
	}
	now := time.Now()
	now = now.In(jst)

	// 環境変数 STATUS_TIMERANGE_TO が設定されている場合はその値をtoとし、設定されていない場合は現在時刻をto, 1ヶ月前をfromとする
	to := os.Getenv("STATUS_TIMERANGE_TO")
	if to == "" {
		to = now.Format("20060102150405")
	}
	parsedTo, err := time.ParseInLocation("20060102150405", to, jst)
	from := parsedTo.AddDate(0, -1, 0).Format("20060102150405")

	getInnerScanRequest := healthplanet.GetStatusRequest{
		DateMode:    healthplanet.DateMode_MeasuredDate,
		From:        from,
		To:          to,
	}
	status, err := hp.GetInnerscan(getInnerScanRequest)
	if err != nil {
		log.Fatal(err)
	}

	for _, data := range status.Data {
		fmt.Println(data.Date)
		fmt.Println(data.KeyData)
		fmt.Println(data.Tag)
		tag, err := hp.GetTagValue(data.Tag)
		if err != nil {
			log.Fatal(err)
		}
		timeJst, _ := time.ParseInLocation("200601021504", data.Date, jst)
		value, _ := strconv.ParseFloat(data.KeyData, 64)
		if value == 0 {
			log.Println("Skipping write due to zero value")
			continue
		}
		err = WriteInfluxDB(influxdbUrl, influxdbToken, influxdbOrg, influxdbBucket, influxdbMeasurement, tag, value, timeJst)
		if err != nil {
			log.Fatalln("Failed to write to InfluxDB:", err)
		}
	}
}
