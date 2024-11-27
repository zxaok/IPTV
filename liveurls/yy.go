package liveurls

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"time"
	"encoding/json"
)

// 处理请求的主要逻辑
func HandleRequest(w http.ResponseWriter, r *http.Request) {
	rid := r.URL.Query().Get("id")

	// 如果没有传递 id 参数，则展示多个预设的直播链接
	if rid == "" {
		// 预设的房间 ID 和名称
		arr := map[string]int{
			"香港赌片[1280*720]":    29900720,
			"英叔全集经典鬼片[1280*720]": 1354869861,
			"|熊猫功夫台":         1355013632,
		}

		// 输出这些直播链接
		for name, id := range arr {
			fmt.Fprintf(w, "%s, http://%s%s?id=%d<br>", name, r.Host, r.URL.Path, id)
		}

		// 获取 YY 歌舞秀信息并展示
		bstrURL := "https://www.yy.com/more/page.action?biz=sing&subBiz=idx&moduleId=308&pageSize=1500"
		data := fetchData(bstrURL)
		re := regexp.MustCompile(`"ssid".*?","desc":"(.*?)"`)
		matches := re.FindAllStringSubmatch(data, -1)

		fmt.Fprintf(w, "===YY歌舞秀===<br>")
		for _, match := range matches {
			fmt.Fprintf(w, "%s, http://%s%s?id=%s<br>", match[1], r.Host, r.URL.Path, match[1])
		}
	} else {
		// 如果传递了 rid 参数，则获取相应的直播流链接
		y := Yy{Rid: rid, Quality: "high"} // 假设质量为高清
		liveUrl := y.GetLiveUrl()

		// 根据设备类型进行不同处理
		if liveUrl != nil {
			if isMobile(r) {
				// 移动端设备，重定向到标清直播流
				playUrl := fmt.Sprintf("http://data.3g.yy.com/live/hls/%s/%s", rid, rid)
				http.Redirect(w, r, playUrl, http.StatusFound)
			} else {
				// 桌面端设备，重定向到高清流 URL
				http.Redirect(w, r, liveUrl.(string), http.StatusFound)
			}
		} else {
			// 没有获取到直播流 URL，返回错误
			http.Error(w, "无法获取直播流", http.StatusNotFound)
		}
	}
}

// 获取直播流地址
type Yy struct {
	Rid     string
	Quality string
}

type StreamLineAddr struct {
	CdnInfo struct {
		Url string `json:"url"`
	} `json:"cdn_info"`
}

type Result struct {
	AvpInfoRes struct {
		StreamLineAddr map[string]StreamLineAddr `json:"stream_line_addr"`
	} `json:"avp_info_res"`
}

func (y *Yy) GetLiveUrl() any {
	firstrid := y.Rid
	quality := y.Quality
	var rid string

	// 检查房间是否存在
	checkUrl := "https://wap.yy.com/mobileweb/" + firstrid
	client := &http.Client{}
	req, _ := http.NewRequest("GET", checkUrl, nil)
	req.Header.Set("Referer", "https://wap.yy.com")
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 16_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.3 Mobile/15E148 Safari/604.1")
	res, _ := client.Do(req)
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	// 解析房间 ID
	re := regexp.MustCompile(`md5Hash[\s\S]*?sid.*'(.*)'.*?getQuery`)
	realdata := re.FindStringSubmatch(string(body))

	if len(realdata) > 0 {
		rid = realdata[1]
	} else {
		return nil
	}

	// 获取时间戳
	millis_13 := time.Now().UnixNano() / int64(time.Millisecond)
	millis_10 := time.Now().Unix()

	// 构造请求参数
	data := fmt.Sprintf(`{"head":{"seq":%d,"appidstr":"0","bidstr":"0","cidstr":"%s","sidstr":"%s","uid64":0,"client_type":108,"client_ver":"5.14.13","stream_sys_ver":1,"app":"yylive_web","playersdk_ver":"5.14.13","thundersdk_ver":"0","streamsdk_ver":"5.14.13"},"client_attribute":{"client":"web","model":"","cpu":"","graphics_card":"","os":"chrome","osversion":"118.0.0.0","vsdk_version":"","app_identify":"","app_version":"","business":"","width":"1728","height":"1117","scale":"","client_type":8,"h265":0},"avp_parameter":{"version":1,"client_type":8,"service_type":0,"imsi":0,"send_time":%d,"line_seq":-1,"gear":%s,"ssl":1,"stream_format":0}}`, millis_13, rid, rid, millis_10, quality)

	// 请求 URL
	url := "https://stream-manager.yy.com/v3/channel/streams?uid=0&cid=" + rid + "&sid=" + rid + "&appid=0&sequence=" + strconv.FormatInt(millis_13, 10) + "&encode=json"
	req, _ = http.NewRequest("POST", url, bytes.NewBuffer([]byte(data)))
	req.Header.Set("Content-Type", "text/plain;charset=UTF-8")
	req.Header.Set("Referer", "https://www.yy.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 Safari/537.36 Edg/106.0.1370.42")

	// 获取响应数据
	res, _ = client.Do(req)
	defer res.Body.Close()
	body, _ = io.ReadAll(res.Body)

	var result Result
	json.Unmarshal(body, &result)

	// 返回直播流 URL
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

// 检测是否为移动设备
func isMobile(r *http.Request) bool {
	userAgent := r.UserAgent()
	mobileKeywords := []string{"iphone", "android", "blackberry", "mobile", "opera mini", "htc", "iemobile"}
	for _, keyword := range mobileKeywords {
		if strings.Contains(strings.ToLower(userAgent), keyword) {
			return true
		}
	}
	return false
}

// 获取网页数据
func fetchData(url string) string {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 Safari/537.36 Edg/106.0.1370.42")
	res, _ := client.Do(req)
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	return string(body)
}
