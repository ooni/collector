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
	"time"

	"github.com/gin-gonic/gin"
)

func mapFromJSON(data []byte) map[string]interface{} {
	var result interface{}
	json.Unmarshal(data, &result)
	return result.(map[string]interface{})
}

func performRequestJSON(r http.Handler, method, path string, reqJSON interface{}) (*httptest.ResponseRecorder, error) {
	body, err := json.Marshal(reqJSON)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w, nil
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
	return performRequestJSON(r, "POST", "/report", createReq)
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
	return performRequestJSON(r, "POST", fmt.Sprintf("/report/%s", reportID), updateReq)
}

func closeReport(r http.Handler, reportID string) (*httptest.ResponseRecorder, error) {
	return performRequestJSON(r, "POST", fmt.Sprintf("/report/%s/close", reportID), nil)
}

func checkDirItemCount(t *testing.T, dirPath string, expected int) {
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != expected {
		t.Errorf("the dir %s does not have %d files as expected (%d instead)", dirPath, len(files), expected)
	}
}

func checkReportIncoming(t *testing.T, ct *CollectorTest, reportID string) {
	path := filepath.Join(ct.IncomingDir(), fmt.Sprintf("%s.json", reportID))
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("report_id file was not created")
	}
}

type CollectorTest struct {
	ReportDir string
	Router    *gin.Engine
}

func (c *CollectorTest) SyncDir() string {
	return filepath.Join(c.ReportDir, "sync")
}

func (c *CollectorTest) IncomingDir() string {
	return filepath.Join(c.ReportDir, "incoming")
}

func NewCollectorTest() (*CollectorTest, error) {
	ct := CollectorTest{}

	reportDir, err := ioutil.TempDir("", "ooni-reports")
	if err != nil {
		return nil, err
	}
	ct.ReportDir = reportDir
	ct.Router = SetupRouter(ct.ReportDir)
	return &ct, err
}

// This test checks to see that if a report is opened and then closed right
// after it doesn't move into the sync directory an empty report file.
func TestReportCreateAndClose(t *testing.T) {
	ct, err := NewCollectorTest()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(ct.ReportDir)

	w, err := createReport(ct.Router)
	if err != nil {
		t.Fatal(err)
	}
	resp := mapFromJSON(w.Body.Bytes())
	reportID := resp["report_id"].(string)

	checkReportIncoming(t, ct, reportID)

	w, err = closeReport(ct.Router, reportID)
	if err != nil {
		t.Fatal(err)
	}

	checkDirItemCount(t, ct.IncomingDir(), 0)
	checkDirItemCount(t, ct.SyncDir(), 0)
}

// This test checks to see that the report lifecycle works fully
func TestReportLifeCycle(t *testing.T) {
	ct, err := NewCollectorTest()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(ct.ReportDir)

	w, err := createReport(ct.Router)
	if err != nil {
		t.Fatal(err)
	}
	resp := mapFromJSON(w.Body.Bytes())
	reportID := resp["report_id"].(string)

	checkReportIncoming(t, ct, reportID)

	wcSample, _ := ioutil.ReadFile("testdata/web_connectivity-sample.json")
	var content interface{}
	json.Unmarshal(wcSample, &content)

	w, err = updateReport(ct.Router, reportID, content)
	if err != nil {
		t.Fatal(err)
	}

	w, err = closeReport(ct.Router, reportID)
	if err != nil {
		t.Fatal(err)
	}

	checkDirItemCount(t, ct.IncomingDir(), 0)
	checkDirItemCount(t, ct.SyncDir(), 1)
}

func TestInvalidFormat(t *testing.T) {
	ct, err := NewCollectorTest()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(ct.ReportDir)

	createReq := CreateReportReq{
		Format: "yaml",
	}
	w, err := performRequestJSON(ct.Router, "POST", "/report", createReq)
	if err != nil {
		t.Error(err)
	}
	if w.Code != 406 {
		t.Error("did not find valid error code")
	}
}

func TestReportsWillExpire(t *testing.T) {
	ct, err := NewCollectorTest()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(ct.ReportDir)

	GetExpiryTimeDuration = func() time.Duration {
		return time.Microsecond
	}

	w, err := createReport(ct.Router)
	if err != nil {
		t.Fatal(err)
	}
	resp := mapFromJSON(w.Body.Bytes())
	reportID := resp["report_id"].(string)
	checkReportIncoming(t, ct, reportID)

	time.Sleep(1 * time.Second)

	checkDirItemCount(t, ct.IncomingDir(), 0)
	checkDirItemCount(t, ct.SyncDir(), 0)
}
