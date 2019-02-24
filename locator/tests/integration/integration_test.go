package integration

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"testing"
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
	appAddr string
)

func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags
	a, _ := getAppAddr()
	appAddr = a
	os.Exit(m.Run())
}

func TestStatus(t *testing.T) {
	resp, err := http.Get("http://" + appAddr + "/v1/status")
	if err != nil {
		t.Fatalf("status failed: %s", err)
	}
	defer resp.Body.Close()

	fmt.Printf("Got status code: %d\n", resp.StatusCode)
	var b bytes.Buffer
	ioutil.ReadAll(&b)
	fmt.Printf("response: %s\n", b.String())
}

func TestSingleInvoke(t *testing.T) {
	var buf bytes.Buffer
	buf.Write([]byte(req1))
	resp, err := http.Post("http://"+appAddr+"/v1/lookup",
		"application/json", &buf)
	if err != nil {
		log.Fatalf("error requesting location: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Unexpected retrun code: %d", resp.StatusCode)
	}

	b, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("here's my response: '%s'\n", b)
}

func getAppAddr() (string, error) {
	res, err := exec.Command("docker-compose", "port", "locator",
		"8080").CombinedOutput()
	if err != nil {
		log.Fatalf("docker-compose error: failed to get exposed port: %v", err)
	}
	return string(res[:len(res)-1]), nil
}
