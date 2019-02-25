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
	"github.com/go-redis/redis"
)

const (
	req1 = `{
    "struct_number": "4600",
    "street": "Silver Hill Rd",
    "city": "Suitland",
    "state": "MD",
    "zip": "20746"
	}`
	req2 = `{
    "struct_number": "204",
    "street": "Williams Ct",
    "city": "Stroudsburg",
    "state": "PA"
	}`

	badReq = `{
    "street": "Silver Hill Rd",
    "city": "Suitland",
    "state": "MD",
    "zip": "20746"
	}`
)

var (
	locatorAddr  string
	analyzerAddr string
	redisAddr    string
	cli          *redis.Client
)

func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags
	locatorAddr, _ = getAppAddr("locator", "8080")
	analyzerAddr, _ = getAppAddr("analyzer", "8090")
	redisAddr, _ = getAppAddr("redis", "6379")
	cli, _ = NewClient()
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
		t.Fatalf("Unexpected return code: %d", resp.StatusCode)
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
		t.Fatalf("error requesting location: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Unexpected return code: %d", resp.StatusCode)
	}
	b, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("here's my response: '%s'\n", b)

	sr := getStatistics()
	fmt.Printf("got statstics: %+v\n", sr)
	if sr.Success != 1 && sr.Error != 0 {
		t.Fatalf("Expected 4 succ, 1 error, but got %d, %d\n", sr.Success, sr.Error)
	}

	s, err := getValueForKey(types.SuccessKey)
	if err != nil {
		t.Fatalf("Key lookup failed: %v", err)
	}
	e, err := getValueForKey(types.ErrorKey)
	if err != redis.Nil {
		t.Fatalf("Key lookup should not have succeeded: %v", err)
	}
	if s != "1" && e != "0" {
		t.Fatalf("Expected 0 succ, 0 error, but got %d, %d\n", sr.Success, sr.Error)
	}
}

func TestConcurrentInvoke(t *testing.T) {
	if err := clearDatabase(); err != nil {
		t.Fatalf("error clearing database: %v", err)
	}
	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			var buf bytes.Buffer
			switch i {
			case 0:
				buf.Write([]byte(badReq))
			case 1:
				buf.Write([]byte(req2))
			default:
				buf.Write([]byte(req1))
			}
			resp, err := http.Post("http://"+locatorAddr+"/v1/lookup",
				"application/json", &buf)
			if err != nil {
				t.Fatalf("error requesting location: %v", err)
			}
			defer resp.Body.Close()

			if i != 0 && resp.StatusCode != http.StatusOK {
				t.Fatalf("Unexpected return code: %d", resp.StatusCode)
			}

			b, _ := ioutil.ReadAll(resp.Body)
			fmt.Printf("got response: '%s'\n", b)
		}()
	}
	wg.Wait()
	sr := getStatistics()
	fmt.Printf("got statstics: %+v\n", sr)
	if sr.Success != 4 && sr.Error != 1 {
		t.Fatalf("Expected 4 succ, 1 error, but got %d, %d\n", sr.Success, sr.Error)
	}

	s, err := getValueForKey(types.SuccessKey)
	if err != nil {
		t.Fatalf("Key lookup failed: %v", err)
	}
	e, err := getValueForKey(types.ErrorKey)
	if err != nil {
		t.Fatalf("Key lookup failed: %v", err)
	}
	if s != "4" && e != "1" {
		t.Fatalf("Expected 4 succ, 1 error, but got %d, %d\n", sr.Success, sr.Error)
	}
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

func NewClient() (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	_, err := client.Ping().Result()
	if err != nil {
		return nil, err
	}
	return client, nil
}

func getValueForKey(key string) (string, error) {
	res, err := cli.Get(key).Result()
	if err != nil {
		return "", err
	}
	return res, nil
}

func clearDatabase() error {
	_, err := http.Get("http://" + analyzerAddr + "/v1/reset")
	return err
}
