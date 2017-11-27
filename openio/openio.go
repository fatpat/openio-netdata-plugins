package openio

import(
    "bufio"
    "strings"
    "path"
    "os"
    "encoding/json"
    "fmt"
    "oionetdata/util"
    "oionetdata/netdata"
)

type serviceType []string

type serviceInfo []struct {
    Addr string
    Score int
}

/*
ProxyURL - Get URL of oioproxy from configuration
*/
func ProxyURL(basePath string, ns string) string {
  file, err := os.Open(path.Join(basePath, ns))
  util.RaiseIf(err)
  defer file.Close()

  scanner := bufio.NewScanner(file)
  for scanner.Scan() {
	t := scanner.Text()
	if strings.HasPrefix(t, "proxy") {
		return strings.Split(t, "=")[1];
	}
  }
  util.RaiseIf(scanner.Err())
  return ""
}

/*
Collect - collect openio metrics
*/
func Collect(proxyURL string, ns string) {
	var sType = serviceTypes(proxyURL, ns)
	for t := range sType  {
		var sInfo = collectScore(proxyURL, ns, sType[t])
		if sType[t] == "rawx" {
			for sc := range sInfo {
				if strings.HasPrefix(sInfo[sc].Addr, strings.Split(proxyURL, ":")[0]) {
					collectRawx(ns, sInfo[sc].Addr)
				}
			}
		} else if strings.HasPrefix(sType[t], "meta") {
			for sc := range sInfo {
				if strings.HasPrefix(sInfo[sc].Addr, strings.Split(proxyURL, ":")[0]) {
					collectMetax(ns, sInfo[sc].Addr, proxyURL)
				}
			}
		}
	}
}

func serviceTypes(proxyURL string, ns string) serviceType {
	url := fmt.Sprintf("http://%s/v3.0/%s/conscience/info?what=types", proxyURL, ns)
	res := serviceType{}
	util.RaiseIf(json.Unmarshal([]byte(util.HTTPGet(url)), &res))
	return res
}

/*
CollectRawx - update metrics for Rawx services
*/
func collectRawx(ns string, service string) {
	url := fmt.Sprintf("http://%s/stat", service)
	var lines = strings.Split(util.HTTPGet(url), "\n");
	for i := range lines {
		s := strings.Split(lines[i], " ")
		if s[0] == "counter" {
			netdata.Update(s[1], sID(service, ns), s[2])
		} else if s[1] == "volume" {
			volumeInfo(service, ns, s[2])
		}
	}
}

/*
CollectMetax - update metrics for M0/M1/M2 servicess
*/
func collectMetax(ns string, service string, proxyURL string) {
	url := fmt.Sprintf("http://%s/v3.0/forward/stats?id=%s", proxyURL, service)
	var lines = strings.Split(util.HTTPGet(url), "\n");
	for i := range lines {
		s := strings.Split(lines[i], " ")
		if s[0] == "counter" {
			netdata.Update(s[1], sID(service, ns), s[2])
		} else if s[1] == "volume" {
            volumeInfo(service, ns, s[2])
		} else if s[0] == "gauge" {
			// TODO: do something with gauge?
		}
	}
}

func volumeInfo(service string, ns string, volume string) {
    for dim, val := range util.VolumeInfo(volume) {
        netdata.Update(dim, sID(service, ns), fmt.Sprint(val))
    }
}

/*
CollectScore - collect score values on all scored services
*/
func collectScore(proxyURL string, ns string, sType string) (serviceInfo) {
	sInfo := serviceInfo{}
	url := fmt.Sprintf("http://%s/v3.0/%s/conscience/list?type=%s", proxyURL, ns, sType)
	util.RaiseIf(json.Unmarshal([]byte(util.HTTPGet(url)), &sInfo))
	for i := range sInfo {
		netdata.Update("score", sID(sInfo[i].Addr, ns), fmt.Sprint(sInfo[i].Score))
	}
	return sInfo
}

func sID(service string, ns string) (string) {
    return fmt.Sprintf("%s.%s", service, ns)
}
