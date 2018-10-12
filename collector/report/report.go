package report

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/apex/log"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/ooni/collector/collector/aws"
	"github.com/ooni/collector/collector/info"
	"github.com/ooni/collector/collector/storage"
	"github.com/ooni/collector/collector/util"
	"github.com/rs/xid"
	"github.com/spf13/viper"
)

// expiryTimers is a map of timers keyed to the ReportID. These are used to
// ensure that after a certain amount of time has elapsed reports are closed
var expiryTimers = make(map[string]*time.Timer)

// expiryTimeDuration is after how much time a report expires
var expiryTimeDuration = time.Duration(8) * time.Hour

// BackendExtra is serverside extra metadata
type BackendExtra struct {
	SubmissionTime time.Time `json:"submission_time" bson:"submission_time"`
	MeasurementID  string    `json:"measurement_id" bson:"measurement_id"`
	ReportID       string    `json:"report_id" bson:"report_id"`
}

// Metadata is metadata about a report
type Metadata struct {
	ReportID        string    `json:"report_id" bson:"report_id"`
	IsClosed        bool      `json:"is_closed" bson:"is_closed"`
	CreationTime    time.Time `json:"creation_time" bson:"creation_time"`
	LastUpdateTime  time.Time `json:"last_update_time" bson:"last_update_time"`
	ProbeASN        string    `json:"probe_asn" bson:"probe_asn"`
	ProbeCC         string    `json:"probe_cc" bson:"probe_cc"`
	Platform        string    `json:"platform" bson:"platform"`
	TestName        string    `json:"test_name" bson:"test_name"`
	SoftwareName    string    `json:"software_name" bson:"software_name"`
	SoftwareVersion string    `json:"software_version" bson:"software_version"`
	EntryCount      int64     `json:"entry_count" bson:"entry_count"`
}

func NewMetadata() *Metadata {
	return &Metadata{}
}

// MeasurementEntry is the structure of measurements submitted by an OONI Probe client
type MeasurementEntry struct {
	// These values are added by the pipeline
	BucketDate     string `json:"bucket_date"`
	ReportFilename string `json:"report_filename"`

	ID                   string       `json:"id"`
	ReportID             string       `json:"report_id"`
	TestName             string       `json:"test_name"`
	TestVersion          string       `json:"test_version"`
	MeasurementStartTime string       `json:"measurement_start_time,omitempty"`
	TestStartTime        string       `json:"test_start_time"` // XXX these should actually be time
	Annotations          interface{}  `json:"annotations"`
	BackendExtra         BackendExtra `json:"backend_extra"`
	BackendVersion       string       `json:"backend_version"`
	DataFormatVersion    string       `json:"data_format_version"`
	Input                string       `json:"input"`
	InputHashes          []string     `json:"input_hashes"`
	Options              []string     `json:"options"`
	ProbeASN             string       `json:"probe_asn"`
	ProbeCC              string       `json:"probe_cc"`
	ProbeCity            string       `json:"probe_city"`
	ProbeIP              string       `json:"probe_ip"`
	SoftwareName         string       `json:"software_name"`
	SoftwareVersion      string       `json:"software_version"`
	TestHelpers          interface{}  `json:"test_helpers"`
	TestKeys             interface{}  `json:"test_keys"`
	TestRuntime          float64      `json:"test_runtime"`
}

// TimestampFormat is the string format for a timestamp, useful for generating
// report ids
const TimestampFormat = "20060102T150405Z"

// GenReportID generates a new report id
func GenReportID(asn string) string {
	return fmt.Sprintf("%s_%s_%s",
		time.Now().UTC().Format(TimestampFormat),
		asn,
		util.RandomStr(50),
	)
}

// CreateNewReport creates a new report
func CreateNewReport(store *storage.Storage, testName string, probeASN string, softwareName string, softwareVersion string) (string, error) {
	reportID := GenReportID(probeASN)
	meta := Metadata{
		ReportID:        reportID,
		TestName:        testName,
		ProbeASN:        probeASN,
		ProbeCC:         "",
		Platform:        "",
		SoftwareName:    softwareName,
		SoftwareVersion: softwareVersion,
		CreationTime:    time.Now().UTC(),
		LastUpdateTime:  time.Now().UTC(),
		IsClosed:        false,
	}
	_, err := store.Client.
		Database("collector").
		Collection("reports").
		InsertOne(context.Background(), meta)
	if err != nil {
		log.WithError(err).Error("failed to allocate a ReportID")
		return "", err
	}
	return meta.ReportID, nil
}

// SQSMessage is the message sent to SQS
type SQSMessage struct {
	ReportID     string
	ReportFile   string
	TestName     string
	CreationTime time.Time
	EntryCount   int64
	CollectorFQN string
}

func sendMessageToSQS(meta *Metadata) {
	message := SQSMessage{
		ReportID:     meta.ReportID,
		TestName:     meta.TestName,
		CreationTime: meta.CreationTime,
		EntryCount:   meta.EntryCount,
		CollectorFQN: viper.GetString("api.fqn"),
	}
	value, err := json.Marshal(message)
	if err != nil {
		log.WithError(err).Error("failed to serialize meta")
		return
	}
	_, err = aws.SendMessage(aws.Session, string(value), "report")
	if err != nil {
		log.WithError(err).Error("failed to publish to aws SQS")
		return
	}
}

func uploadToS3(meta *Metadata) {
	/*
		XXX
			filename := filepath.Base(meta.ReportFilePath)
			bucket := viper.GetString("aws.s3-bucket")
			prefix := fmt.Sprintf("%s/%s",
				viper.GetString("aws.s3-prefix"),
				time.Now().UTC().Format("2006-01-02"))
			// We place files inside the directory $PREFIX/$YEAR-$MONTH-$DAY/
			key := fmt.Sprintf("%s/%s", prefix, filename)
			err := aws.UploadFile(aws.Session, meta.ReportFilePath, bucket, key)
			if err != nil {
				log.WithError(err).Errorf("failed to upload to s3://%s/%s", bucket, key)
				return
			}
	*/
}

func performAWSTasks(meta *Metadata) {
	sendMessageToSQS(meta)
	uploadToS3(meta)
}

// CloseReport marks the report as closed and moves it into the final reports folder
func CloseReport(store *storage.Storage, reportID string) error {
	var err error

	meta := NewMetadata()
	reportFilter := bson.NewDocument(bson.EC.String("report_id", reportID))
	err = store.Client.
		Database("collector").
		Collection("reports").
		FindOne(context.Background(), reportFilter).
		Decode(meta)
	if err != nil {
		log.WithError(err).Error("failed to find report_id")
		return err
	}
	if meta.IsClosed == true {
		return ErrReportIsClosed
	}

	_, err = store.Client.
		Database("collector").
		Collection("measurements").
		UpdateMany(
			context.Background(),
			reportFilter,
			bson.EC.SubDocumentFromElements("$set",
				bson.EC.Boolean("is_closed", true),
			),
		)
	if err != nil {
		log.WithError(err).Error("failed to update measurements with is_closed=true")
		return err
	}

	meta.IsClosed = true
	_, err = store.Client.
		Database("collector").
		Collection("reports").
		UpdateOne(nil, reportFilter, meta)
	if err != nil {
		log.WithError(err).Error("failed to update report with is_closed=true")
		return err
	}

	if aws.Session != nil {
		go performAWSTasks(meta)
	}

	return nil
}

func genMeasurementID() string {
	return xid.New().String()
}

func validateMetadata(meta *Metadata, entry *MeasurementEntry) error {
	return nil
}

func addBackendExtra(meta *Metadata, entry *MeasurementEntry) string {
	measurementID := genMeasurementID()
	entry.BackendVersion = info.Version
	entry.BackendExtra.SubmissionTime = meta.LastUpdateTime
	entry.BackendExtra.ReportID = meta.ReportID
	entry.BackendExtra.MeasurementID = measurementID
	return measurementID
}

var probeCCRegexp = regexp.MustCompile("^[A-Z]{2}$")

// WriteEntry will write an entry to report
func WriteEntry(store *storage.Storage, reportID string, entry *MeasurementEntry) (string, error) {
	var err error

	meta := NewMetadata()
	reportFilter := bson.NewDocument(bson.EC.String("report_id", reportID))
	err = store.Client.
		Database("collector").
		Collection("reports").
		FindOne(context.Background(), reportFilter).
		Decode(meta)
	if err != nil {
		return "", err
	}

	if meta.IsClosed == true {
		return "", ErrReportIsClosed
	}

	// If the ProbeCC or Platform are the empty string it means it's the first
	// entry so we should parse this from the measurement entry and add it to the
	// metadata.
	if meta.ProbeCC == "" {
		if probeCCRegexp.MatchString(entry.ProbeCC) != true {
			return "", errors.New("Invalid probe_cc")
		}
		meta.ProbeCC = entry.ProbeCC
	}
	if meta.Platform == "" && entry.Annotations != nil {
		annotations := entry.Annotations.(map[string]interface{})
		platform, ok := annotations["platform"].(string)
		if ok {
			meta.Platform = platform
		}
	}

	err = validateMetadata(meta, entry)
	if err != nil {
		log.WithError(err).Error("inconsistent metadata found")
		return "", err
	}
	measurementID := addBackendExtra(meta, entry)
	entryBytes, err := json.Marshal(entry)
	if err != nil {
		log.WithError(err).Error("could not serialize entry")
		return "", err
	}

	_, err = store.Client.
		Database("collector").
		Collection("measurement_entries").
		InsertOne(
			context.Background(),
			bson.NewDocument(
				bson.EC.String("report_id", entry.ReportID),
				bson.EC.String("measurement_id", measurementID),
				bson.EC.Binary("json_bytes", entryBytes),
			),
		)
	if err != nil {
		log.WithError(err).Error("failed to insert into measurements table")
		return "", err
	}

	meta.LastUpdateTime = time.Now().UTC()
	meta.EntryCount++

	_, err = store.Client.
		Database("collector").
		Collection("reports").
		UpdateOne(
			nil,
			reportFilter,
			bson.NewDocument(
				bson.EC.SubDocumentFromElements("$set",
					bson.EC.DateTime("last_update_time",
						meta.LastUpdateTime.UnixNano()/int64(time.Millisecond)),
					bson.EC.Int64("entry_count", meta.EntryCount),
					// XXX this is a bit of a silly thing to do on every entry
					bson.EC.String("probe_cc", meta.ProbeCC),
					bson.EC.String("platform", meta.Platform),
				),
			),
		)
	if err != nil {
		log.WithError(err).Error("failed to update reports table")
		return "", err
	}
	return measurementID, nil
}
