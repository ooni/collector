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

type UpdateReportReq struct {
	Content interface{} `json:"content"`
	Format  string      `json:"format"`
}

func updateReport(r http.Handler, reportID string, content interface{}) (*httptest.ResponseRecorder, error) {
	updateReq := UpdateReportReq{
		Format:  "json",
		Content: content,
	}
	body, err := json.Marshal(updateReq)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("/report/%s", reportID), bytes.NewReader(body))
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

// This test checks to see that if a report is opened and then closed right
// after it doesn't move into the sync directory an empty report file.
func TestReportCreateAndClose(t *testing.T) {
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

	if _, err := os.Stat(filepath.Join(incomingDirPath(reportDir), fmt.Sprintf("%s.json", reportID))); os.IsNotExist(err) {
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

// This test checks to see that the report lifecycle works fully
func TestReportLifeCycle(t *testing.T) {
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

	if _, err := os.Stat(filepath.Join(incomingDirPath(reportDir), fmt.Sprintf("%s.json", reportID))); os.IsNotExist(err) {
		t.Error("report_id file was not created")
	}

	wcSample, _ := ioutil.ReadFile("testdata/web_connectivity-sample.json")
	var content interface{}
	json.Unmarshal(wcSample, &content)

	w, err = updateReport(router, reportID, content)
	if err != nil {
		t.Fatal(err)
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
	if len(files) != 1 {
		t.Error("the sync dir is empty!")
	}
}
