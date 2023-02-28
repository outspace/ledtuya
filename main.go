package main

import (
    "bytes"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "sort"
    "strings"
    "time"
    "sync"
    "image/color"
    "github.com/PerformLine/go-stockutil/colorutil"
    "net/http/httputil"
    "github.com/cenkalti/dominantcolor"
    "github.com/kbinani/screenshot"
)


var (
    Token string 
	Host = "https://openapi.tuyaeu.com"
    ClientID = "__"
    Secret = "___"
    //PUT your devices in this array 
    // EXAMPLE:
    // (one dev) DeviceID = [...]string{"bf21c1b7582f28e0bf8hgz"}
    // (two dev) DeviceID = [...]string{"bfb9f75c7fcccaa01elmox", "bf21c1b7582f28e0bf8hgz"}
    // (n dev)  DeviceID = [...]string{"bfb9f75c7fcccaa01elmox", "bf21c1b7582f28e0bf8hgz", ..(n - 3).., "aaaaaaaaaaaaaaaaaa"}
    DeviceID = [...] string { "bfb9f75c7fcccaa01elmox", "bf21c1b7582f28e0bf8hgz"}
)

type TokenResponse struct {
	Result struct {
		AccessToken  string `json:"access_token"`
		ExpireTime   int    `json:"expire_time"`
		RefreshToken string `json:"refresh_token"`
		UID          string `json:"uid"`
	} `json:"result"`
	Success bool  `json:"success"`
	T       int64 `json:"t"`
}

func main() {
    GetToken()
    inputControl()
}

func GetToken() {
	method := "GET"
	body := []byte(``)
	req, _ := http.NewRequest(method, Host+"/v1.0/token?grant_type=1", bytes.NewReader(body))

	buildHeader(req, body)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)
		return
	}
	defer resp.Body.Close()
	bs, _ := ioutil.ReadAll(resp.Body)
	ret := TokenResponse{}
	json.Unmarshal(bs, &ret)
	log.Println("resp:", string(bs))

	if v := ret.Result.AccessToken; v != "" {
		Token = v
	}
}

// formatRequest generates ascii representation of a request
func formatRequest(r *http.Request) string {
 // Create return string
 var request []string // Add the request string
 url := fmt.Sprintf("%v %v %v", r.Method, r.URL, r.Proto)
 request = append(request, url) // Add the host
 request = append(request, fmt.Sprintf("Host: %v", r.Host)) // Loop through headers
 for name, headers := range r.Header {
   name = strings.ToLower(name)
   for _, h := range headers {
     request = append(request, fmt.Sprintf("%v: %v", name, h))
   }
 }
 
 // If this is a POST, add post data
 if r.Method == "POST" {
    r.ParseForm()
    request = append(request, "\n")
    request = append(request, r.Form.Encode())
 }   // Return the request as a string
  return strings.Join(request, "\n")
}


// Function to translate named color RGBA to RGB, preserving some A
func rgba2rgb(incolor color.RGBA)(float64, float64, float64) {
    var alpha = incolor.A

    return float64(255 / alpha * incolor.R),
        float64(255 / alpha * incolor.G),
        float64(255 / alpha * incolor.B)
}

func inputControl() {
    for {
        bounds := screenshot.GetDisplayBounds(0)

        img, err := screenshot.CaptureRect(bounds)
        if err != nil {
            panic(err)
        }

        color := color.RGBA(dominantcolor.Find(img))

        hslH, hslS, hslV := colorutil.RgbToHsl(rgba2rgb(color))
        c := make(chan string)

		var wg sync.WaitGroup
		wg.Add(len(DeviceID))
		for ii := 0; ii < len(DeviceID); ii++ {
			go func(c chan string) {
				for {
					device, more := <-c
					if more == false {
						wg.Done()
						return
					}
					sendChangeColorRequest(device, hslH, hslS, hslV)
				}
			}(c)
		}
		for _, a := range DeviceID {
			c <- a
		}
		close(c)
		wg.Wait()

        time.Sleep(500 * time.Millisecond)
    }
}

func sendChangeColorRequest(deviceId string, H float64, S float64, V float64) {
    hslS := S * 1000
    hslV := V * 1000

    body := [] byte(fmt.Sprintf(`{
				"commands": [
				  {
					"code": "colour_data_v2",
					"value": {"h":%d,"s":%d,"v":%d}
				  }
				]
			  }`, int(H), int(hslS), int(hslV)))
    req, _ := http.NewRequest(http.MethodPost, Host + "/v1.0/iot-03/devices/" + deviceId + "/commands", bytes.NewReader(body))

    buildHeader(req, body)

    log.Println("req:", formatRequest(req))
    reqDump, err := httputil.DumpRequestOut(req, true)
    if err != nil {
        log.Fatal(err)
    }

    log.Println("REQUEST:\n%s", string(reqDump))

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        log.Println(err)
        return
    }
    defer resp.Body.Close()
    bs, _ := ioutil.ReadAll(resp.Body)
    log.Println("resp:", string(bs))

}

func buildHeader(req * http.Request, body[] byte) {
    req.Header.Set("client_id", ClientID)
    req.Header.Set("sign_method", "HMAC-SHA256")

    ts := fmt.Sprint(time.Now().UnixNano() / 1e6)
    req.Header.Set("t", ts)

    if Token != "" {
        req.Header.Set("access_token", Token)
    }

    sign := buildSign(req, body, ts)
    req.Header.Set("sign", sign)
}

func buildSign(req * http.Request, body[] byte, t string) string {
    headers := getHeaderStr(req)
    urlStr := getUrlStr(req)
    contentSha256 := Sha256(body)
    stringToSign := req.Method + "\n" + contentSha256 + "\n" + headers + "\n" + urlStr
    signStr := ClientID + Token + t + stringToSign
    sign := strings.ToUpper(HmacSha256(signStr, Secret))
    return sign
}

func Sha256(data[] byte) string {
    sha256Contain := sha256.New()
    sha256Contain.Write(data)
    return hex.EncodeToString(sha256Contain.Sum(nil))
}

func getUrlStr(req *http.Request) string {
	url := req.URL.Path
	keys := make([]string, 0, 10)

	query := req.URL.Query()
	for key, _ := range query {
		keys = append(keys, key)
	}
	if len(keys) > 0 {
		url += "?"
		sort.Strings(keys)
		for _, keyName := range keys {
			value := query.Get(keyName)
			url += keyName + "=" + value + "&"
		}
	}

	if url[len(url)-1] == '&' {
		url = url[:len(url)-1]
	}
	return url
}

func getHeaderStr(req *http.Request) string {
	signHeaderKeys := req.Header.Get("Signature-Headers")
	if signHeaderKeys == "" {
		return ""
	}
	keys := strings.Split(signHeaderKeys, ":")
	headers := ""
	for _, key := range keys {
		headers += key + ":" + req.Header.Get(key) + "\n"
	}
	return headers
}

func HmacSha256(message string, secret string) string {
	key := []byte(secret)
	h := hmac.New(sha256.New, key)
	h.Write([]byte(message))
	sha := hex.EncodeToString(h.Sum(nil))
	return sha
}
