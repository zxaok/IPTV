package liveurls

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

// Yy struct encapsulates the live URL logic
type Yy struct {
	Rid     string
	Quality string
}

// StreamLineAddr struct for holding URL information from JSON
type StreamLineAddr struct {
	CdnInfo struct {
		Url string `json:"url"`
	} `json:"cdn_info"`
}

// Result struct for parsing the JSON response
type Result struct {
	AvpInfoRes struct {
		StreamLineAddr map[string]StreamLineAddr `json:"stream_line_addr"`
	} `json:"avp_info_res"`
}

// GetLiveUrl fetches and returns the live stream URL for the given `rid` and `quality`
func (y *Yy) GetLiveUrl() interface{} {
	firstrid := y.Rid
	quality := y.Quality
	var rid string
	checkUrl := "https://wap.yy.com/mobileweb/" + firstrid

	client := &http.Client{}
	req, _ := http.NewRequest("GET", checkUrl, nil)
	req.Header.Set("Referer", "https://wap.yy.com")
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 16_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.3 Mobile/15E148 Safari/604.1")

	res, _ := client.Do(req)
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	re := regexp.MustCompile(`md5Hash[\s\S]*?sid.*'(.*)'.*?getQuery`)
	realdata := re.FindStringSubmatch(string(body))

	if len(realdata) > 0 {
		rid = realdata[1]
	} else {
		return nil
	}

	// Prepare the data for the API call
	millis_13 := time.Now().UnixNano() / int64(time.Millisecond)
	millis_10 := time.Now().Unix()
	data := fmt.Sprintf(`{"head":{"seq":%d,"appidstr":"0","bidstr":"0","cidstr":"%s","sidstr":"%s","uid64":0,"client_type":108,"client_ver":"5.14.13","stream_sys_ver":1,"app":"yylive_web","playersdk_ver":"5.14.13","thundersdk_ver":"0","streamsdk_ver":"5.14.13"},"client_attribute":{"client":"web","model":"","cpu":"","graphics_card":"","os":"chrome","osversion":"118.0.0.0","vsdk_version":"","app_identify":"","app_version":"","business":"","width":"1728","height":"1117","scale":"","client_type":8,"h265":0},"avp_parameter":{"version":1,"client_type":8,"service_type":0,"imsi":0,"send_time":%d,"line_seq":-1,"gear":%s,"ssl":1,"stream_format":0}}`, millis_13, rid, rid, millis_10, quality)

	url := "https://stream-manager.yy.com/v3/channel/streams?uid=0&cid=" + rid + "&sid=" + rid + "&appid=0&sequence=" + strconv.FormatInt(millis_13, 10) + "&encode=json"
	req, _ = http.NewRequest("POST", url, bytes.NewBuffer([]byte(data)))
	req.Header.Set("Content-Type", "text/plain;charset=UTF-8")
	req.Header.Set("Referer", "https://www.yy.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 Safari/537.36 Edg/106.0.1370.42")

	res, _ = client.Do(req)
	defer res.Body.Close()
	body, _ = io.ReadAll(res.Body)

	// Parse the result and get the stream URL
	var result Result
	json.Unmarshal(body, &result)
	if len(result.AvpInfoRes.StreamLineAddr) > 0 {
		var arr []string
		for k := range result.AvpInfoRes.StreamLineAddr {
			arr = append(arr, k)
		}
		return result.AvpInfoRes.StreamLineAddr[arr[0]].CdnInfo.Url
	} else {
		return nil
	}
}

// Helper function to check if the request is from a mobile device
func isMobile(r *http.Request) bool {
	userAgent := r.Header.Get("User-Agent")
	return strings.Contains(userAgent, "Mobile") || strings.Contains(userAgent, "Android") || strings.Contains(userAgent, "iPhone")
}

// Handler function to process incoming requests
func liveUrlHandler(w http.ResponseWriter, r *http.Request) {
	rid := r.URL.Query().Get("id")
	if rid == "" {
		// Handle the case where no ID is provided
		fmt.Fprintf(w, "Missing ID parameter")
		return
	}

	// Create a Yy instance and fetch the live URL
	yy := &Yy{Rid: rid, Quality: "high"}
	liveUrl := yy.GetLiveUrl()

	// If mobile, redirect to mobile version
	if isMobile(r) {
		http.Redirect(w, r, liveUrl.(string), http.StatusFound)
		return
	}

	// Handle desktop requests by returning the live URL
	fmt.Fprintf(w, "Live URL: %s", liveUrl)
}

func main() {
	http.HandleFunc("/live", liveUrlHandler)
	http.ListenAndServe(":8080", nil)
}
