package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"testing"

	"github.com/gdotgordon/locator-demo/analyzer/types"
)

const (
	req1 = `{
    "struct_number": "4600",
    "street": "Silver Hill Rd",
    "city": "Suitland",
    "state": "MD",
    "zip": "20746"
}`
)

var (
	locatorAddr  string
	analyzerAddr string
)

func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags
	locatorAddr, _ = getAppAddr("locator", "8080")
	analyzerAddr, _ = getAppAddr("analyzer", "8090")
	fmt.Println("locator", locatorAddr, "analyzer", analyzerAddr)
	os.Exit(m.Run())
}

func TestStatus(t *testing.T) {
	if err := clearDatabase(); err != nil {
		t.Fatalf("error clearing database: %v", err)
	}
	resp, err := http.Get("http://" + locatorAddr + "/v1/status")
	if err != nil {
		t.Fatalf("status failed: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Unexpected return code: %d", resp.StatusCode)
	}
	var b bytes.Buffer
	ioutil.ReadAll(&b)
	fmt.Printf("response: %s\n", b.String())
}

func TestSingleInvoke(t *testing.T) {
	if err := clearDatabase(); err != nil {
		t.Fatalf("error clearing database: %v", err)
	}
	var buf bytes.Buffer
	buf.Write([]byte(req1))
	resp, err := http.Post("http://"+locatorAddr+"/v1/lookup",
		"application/json", &buf)
	if err != nil {
		log.Fatalf("error requesting location: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Unexpected return code: %d", resp.StatusCode)
	}
	b, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("here's my response: '%s'\n", b)

	sr := getStatistics()
	fmt.Printf("got statstics: %+v\n", sr)
}

func TestConcurrentInvoke(t *testing.T) {
	if err := clearDatabase(); err != nil {
		t.Fatalf("error clearing database: %v", err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var buf bytes.Buffer
			buf.Write([]byte(req1))
			resp, err := http.Post("http://"+locatorAddr+"/v1/lookup",
				"application/json", &buf)
			if err != nil {
				log.Fatalf("error requesting location: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				log.Fatalf("Unexpected return code: %d", resp.StatusCode)
			}

			b, _ := ioutil.ReadAll(resp.Body)
			fmt.Printf("here's my response: '%s'\n", b)
		}()
	}
	wg.Wait()
	sr := getStatistics()
	fmt.Printf("got statstics: %+v\n", sr)
}

func getAppAddr(app, port string) (string, error) {
	res, err := exec.Command("docker-compose", "port", app, port).CombinedOutput()
	if err != nil {
		log.Fatalf("docker-compose error: failed to get exposed port: %v", err)
	}
	return string(res[:len(res)-1]), nil
}

func getStatistics() types.StatsResponse {
	resp, err := http.Get("http://" + analyzerAddr + "/v1/statistics")
	if err != nil {
		log.Fatalf("error requesting statistics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Unexpected return code: %d", resp.StatusCode)
	}

	var sr types.StatsResponse
	if err = json.NewDecoder(resp.Body).Decode(&sr); err != nil {

	}
	return sr
}

func clearDatabase() error {
	_, err := http.Get("http://" + analyzerAddr + "/v1/reset")
	return err
}
