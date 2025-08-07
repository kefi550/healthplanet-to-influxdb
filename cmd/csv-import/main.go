package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

var (
	influxdbUrl         = os.Getenv("INFLUXDB_URL")
	influxdbToken       = os.Getenv("INFLUXDB_TOKEN")
	influxdbOrg         = os.Getenv("INFLUXDB_ORG")
	influxdbBucket      = os.Getenv("INFLUXDB_BUCKET")
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
	var csvFile string
	var dryRun bool
	
	flag.StringVar(&csvFile, "csv", "", "CSV file path (required)")
	flag.StringVar(&csvFile, "f", "", "CSV file path (short form)")
	flag.BoolVar(&dryRun, "dry-run", false, "Dry run mode - don't actually write to InfluxDB")
	flag.Parse()

	if csvFile == "" {
		fmt.Println("Usage: csv-import -csv <file.csv> [-dry-run]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// 環境変数チェック
	if influxdbUrl == "" || influxdbToken == "" || influxdbOrg == "" || influxdbBucket == "" {
		log.Fatal("Required environment variables not set: INFLUXDB_URL, INFLUXDB_TOKEN, INFLUXDB_ORG, INFLUXDB_BUCKET")
	}

	// CSVファイルを開く
	file, err := os.Open(csvFile)
	if err != nil {
		log.Fatalf("Failed to open CSV file: %v", err)
	}
	defer file.Close()

	// CSV reader作成
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("Failed to read CSV: %v", err)
	}

	if len(records) == 0 {
		log.Fatal("CSV file is empty")
	}

	fmt.Printf("Reading CSV file: %s\n", csvFile)
	
	// ヘッダー行をチェック
	if len(records) < 2 {
		log.Fatal("CSV file must have at least header and data rows")
	}

	// データ行を処理（ヘッダー行をスキップ）
	count := 0
	errors := 0
	for i, record := range records[1:] {
		// 空行をスキップ
		if len(record) == 0 || (len(record) == 1 && strings.TrimSpace(record[0]) == "") {
			continue
		}

		if len(record) < 3 {
			log.Printf("Skipping row %d: insufficient columns (expected 3, got %d)", i+2, len(record))
			errors++
			continue
		}

		timeStr := strings.TrimSpace(record[0])
		tag := strings.TrimSpace(record[1])
		fieldStr := strings.TrimSpace(record[2])

		// フィールド値を数値に変換
		value, err := strconv.ParseFloat(fieldStr, 64)
		if err != nil {
			log.Printf("Skipping row %d: invalid field value '%s': %v", i+2, fieldStr, err)
			errors++
			continue
		}

		// ゼロ値をスキップ（既存のロジックに合わせる）
		if value == 0 {
			log.Printf("Skipping row %d: zero value", i+2)
			continue
		}

		// タイムスタンプ解析
		timestamp, err := parseTimestamp(timeStr)
		if err != nil {
			log.Printf("Skipping row %d: failed to parse timestamp '%s': %v", i+2, timeStr, err)
			errors++
			continue
		}

		measurement := influxdbMeasurement
		if dryRun {
			fmt.Printf("DRY RUN: Would write - measurement=%s, tag=%s, value=%f, time=%s\n", 
				measurement, tag, value, timestamp.Format(time.RFC3339))
		} else {
			err = WriteInfluxDB(influxdbUrl, influxdbToken, influxdbOrg, influxdbBucket, measurement, tag, value, timestamp)
			if err != nil {
				log.Printf("Failed to write row %d to InfluxDB: %v", i+2, err)
				errors++
				continue
			}
		}
		count++
	}

	fmt.Printf("Successfully processed %d records", count)
	if errors > 0 {
		fmt.Printf(" (%d errors)", errors)
	}
	fmt.Println()
}

func parseTimestamp(timestampStr string) (time.Time, error) {
	// 複数のタイムスタンプ形式をサポート
	formats := []string{
		time.RFC3339,           // 2020-01-01T00:00:00Z
		time.RFC3339Nano,       // 2020-01-01T00:00:00.000Z
		"2006-01-02T15:04:05",  // 2020-01-01T00:00:00
		"2006-01-02 15:04:05",  // 2020-01-01 00:00:00
		"200601021504",         // 202001010000 (HealthPlanet format)
		"20060102150405",       // 20200101000000
		"2006-01-02",           // 2020-01-01
		"2006/01/02 15:04:05",  // 2020/01/01 00:00:00
		"2006/01/02",           // 2020/01/01
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timestampStr); err == nil {
			return t, nil
		}
	}

	// Unix timestampとして解析を試行
	if unix, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
		// 秒単位かミリ秒単位かを判定
		if unix > 1e12 { // ミリ秒
			return time.Unix(0, unix*int64(time.Millisecond)), nil
		} else { // 秒
			return time.Unix(unix, 0), nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse timestamp: %s", timestampStr)
}
