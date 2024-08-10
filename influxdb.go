package hp2influxdb

import (
	"context"
	"fmt"
	"time"

	"github.com/influxdata/influxdb-client-go/v2"
)

func WriteInfluxDB(host, token, org, bucket, measurement string, tag string, value float64, t time.Time) error {
	client := influxdb2.NewClient(host, token)
	writeAPI := client.WriteAPIBlocking(org, bucket)

	fmt.Println(t)

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
