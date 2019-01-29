package collector

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/apex/log"
)

func syncDirPath(reportDir string) string {
	return filepath.Join(reportDir, "sync")
}

func incomingDirPath(reportDir string) string {
	return filepath.Join(reportDir, "incoming")
}

func mapFromJSON(data []byte) map[string]interface{} {
	var result interface{}
	json.Unmarshal(data, &result)
	return result.(map[string]interface{})
}

type CreateReportReq struct {
	SoftwareName      string `json:"software_name"`
	SoftwareVersion   string `json:"software_version"`
	ProbeASN          string `json:"probe_asn"`
	ProbeCC           string `json:"probe_cc"`
	TestName          string `json:"test_name"`
	TestVersion       string `json:"test_version"`
	DataFormatVersion string `json:"data_format_version"`
	Format            string `json:"format"`
	// The below fields are optional
	TestStartTime string   `json:"test_start_time,omitempty"`
	InputHashes   []string `json:"input_hashes,omitempty"`
	TestHelper    string   `json:"test_helper,omitempty"`
	Content       string   `json:"content,omitempty"`
	ProbeIP       string   `json:"probe_ip,omitempty"`
}

func createReport(r http.Handler) (*httptest.ResponseRecorder, error) {
	createReq := CreateReportReq{
		SoftwareName:      "collector-tester",
		SoftwareVersion:   "0.0.1-dev",
		ProbeASN:          "AS1234",
		ProbeCC:           "IT",
		TestName:          "collector_experiment",
		DataFormatVersion: "0.2.0",
		Format:            "json",
	}
	body, err := json.Marshal(createReq)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", "/report", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w, nil
}

func closeReport(r http.Handler, reportID string) (*httptest.ResponseRecorder, error) {
	req, err := http.NewRequest("POST", fmt.Sprintf("/report/%s/close", reportID), nil)
	if err != nil {
		return nil, err
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w, nil
}

func TestReportCreate(t *testing.T) {
	reportDir, err := ioutil.TempDir("", "ooni-reports")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(reportDir)

	router := SetupRouter(reportDir)
	w, err := createReport(router)
	if err != nil {
		t.Fatal(err)
	}
	resp := mapFromJSON(w.Body.Bytes())
	reportID := resp["report_id"].(string)

	if _, err := os.Stat(filepath.Join(incomingDirPath(reportDir), reportID)); os.IsNotExist(err) {
		t.Error("report_id file was not created")
	}

	w, err = closeReport(router, reportID)
	if err != nil {
		t.Fatal(err)
	}

	files, err := ioutil.ReadDir(incomingDirPath(reportDir))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Error("the incoming dir is not empty")
	}
	files, err = ioutil.ReadDir(syncDirPath(reportDir))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		log.Error("the sync dir is not empty. Empty files should not be moved into the sync dir")
	}

}
