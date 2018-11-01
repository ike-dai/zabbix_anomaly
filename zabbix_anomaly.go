package main

import (
	"flag"
	"fmt"
	"github.com/AlekSi/zabbix"
	"github.com/AlekSi/zabbix-sender"
	anomalydetector "github.com/chobie/go-anomalydetector"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type ChangeFinder struct {
	O         *anomalydetector.AnomalyDetector
	S         *anomalydetector.AnomalyDetector
	Smooth    int
	Last      float64
	LastScore float64
	Buffer    []float64
	Buffer2   []float64
}

func sum(array *[]float64) float64 {
	sum := 0.0
	for _, value := range *array {
		sum += value
	}
	return sum
}

func (finder *ChangeFinder) Update(v float64) float64 {
	var score float64 = 0.0

	if v == finder.Last && finder.LastScore < 3.0 {
		score = finder.LastScore
	} else {
		r := finder.O.Update(v)
		finder.Buffer = append(finder.Buffer, r)

		if len(finder.Buffer) > finder.Smooth {
			finder.Buffer = finder.Buffer[1:]
		}

		score = finder.S.Update(sum(&finder.Buffer) / float64(len(finder.Buffer)))
	}

	finder.Last = v
	finder.LastScore = score

	return finder.LastScore
}

func NewChangePoint(outlier_term int, outlier_discount float64, score_term int, score_discount float64, smooth_term int) *ChangeFinder {
	v := ChangeFinder{}
	v.O = anomalydetector.NewAnomalyDetector(outlier_term, outlier_discount)
	v.S = anomalydetector.NewAnomalyDetector(score_term, score_discount)
	v.Smooth = smooth_term

	return &v
}

func main() {
	var zabbix_host, zabbix_port, zabbix_api_url, zabbix_user, zabbix_password, itemid string
	var value_type, host_name, orig_item_key, orig_item_delay string
	var interval, from_time, num, sec_multiplier int64
	var send_data []zabbix_sender.DataItem
	var outlier_term, score_term, smooth_term int
	var outlier_discount, score_discount float64
	var item_prefix string
	// Set Option
	flag.StringVar(&zabbix_host, "host", "localhost", "Set zabbix host name")
	flag.StringVar(&zabbix_host, "h", "localhost", "Set zabbix host name")
	flag.StringVar(&zabbix_port, "port", "10051", "Set zabbix host port")
	flag.StringVar(&zabbix_port, "p", "10051", "Set zabbix host port")
	flag.StringVar(&zabbix_api_url, "url", "http://"+zabbix_host+"/zabbix/api_jsonrpc.php", "Set zabbix api url")
	flag.StringVar(&zabbix_api_url, "u", "http://"+zabbix_host+"/zabbix/api_jsonrpc.php", "Set zabbix api url")
	flag.StringVar(&zabbix_user, "user", "Admin", "Set zabbix login username")
	flag.StringVar(&zabbix_password, "pass", "zabbix", "Set zabbix login user password")
	flag.StringVar(&zabbix_password, "password", "zabbix", "Set zabbix login user password")
	flag.StringVar(&itemid, "itemid", "10000", "Set target zabbix item id")
	flag.StringVar(&itemid, "i", "10000", "Set target zabbix item id")
	flag.Int64Var(&interval, "interval", 300, "Set monitoring interval")
	flag.IntVar(&outlier_term, "outlier_term", 5, "Set outlier_term num")
	flag.IntVar(&score_term, "score_term", 5, "Set score_term num")
	flag.IntVar(&smooth_term, "smooth_term", 5, "Set smooth_term num")
	flag.Float64Var(&outlier_discount, "outlier_discount", 0.02, "Set outlier_discount value")
	flag.Float64Var(&score_discount, "score_discount", 0.02, "Set score_discount value")
	flag.Int64Var(&num, "num", 30, "Set evaluation datapoint num")
	flag.Int64Var(&num, "n", 30, "Set evaluation datapoint num")
	flag.StringVar(&item_prefix, "prefix", "anomaly", "Set registered trapper item key prefix")
	flag.Parse()

	api := zabbix.NewAPI(zabbix_api_url)
	_, err := api.Login(zabbix_user, zabbix_password)
	if err != nil {
		fmt.Printf("[ERROR]: zabbix api login error: %s", err)
		os.Exit(1)
	}
	now := time.Now().Unix()

	// Get orig item info
	item_response, err := api.Call("item.get", zabbix.Params{"itemids": itemid, "selectHosts": "extend", "output": "extend"})
	if err != nil {
		fmt.Printf("[ERROR]: zabbix api item.get error: %s", err)
		os.Exit(1)
	}
	for _, item := range item_response.Result.([]interface{}) {
		value_type = item.(map[string]interface{})["value_type"].(string)
		orig_item_key = item.(map[string]interface{})["key_"].(string)
		orig_item_delay = item.(map[string]interface{})["delay"].(string)

		// for support Time suffixes feature over Zabbix 3.4.0
		sec_multiplier = 1
		if strings.HasSuffix(orig_item_delay, "m") {
			sec_multiplier = 60
			orig_item_delay = orig_item_delay[:len(orig_item_delay)-1]
		} else if strings.HasSuffix(orig_item_delay, "h") {
			sec_multiplier = 60 * 60
			orig_item_delay = orig_item_delay[:len(orig_item_delay)-1]
		} else if strings.HasSuffix(orig_item_delay, "d") {
			sec_multiplier = 60 * 60 * 24
			orig_item_delay = orig_item_delay[:len(orig_item_delay)-1]
		} else if strings.HasSuffix(orig_item_delay, "w") {
			sec_multiplier = 60 * 60 * 24 * 7
			orig_item_delay = orig_item_delay[:len(orig_item_delay)-1]
		}
		int64_delay_count, _ := strconv.ParseInt(orig_item_delay, 10, 64)
		int64_delay := int64_delay_count * sec_multiplier
		from_time = now - int64_delay*num

		for _, host := range item.(map[string]interface{})["hosts"].([]interface{}) {
			host_name = host.(map[string]interface{})["host"].(string)
		}
	}

	// Get Zabbix History
	response, _ := api.Call("history.get", zabbix.Params{"history": value_type, "itemids": itemid, "time_from": from_time, "time_till": now, "output": "extend"})
	//cp := NewChangePoint(12, 0.0275, 6, 0.1, 12)
	//cp := NewChangePoint(5, 0.02, 5, 0.02, 5)
	cp := NewChangePoint(outlier_term, outlier_discount, score_term, score_discount, smooth_term)
	//cp := NewChangePoint(7, 0.5, 28, 0.01, 7)
	for _, history := range response.Result.([]interface{}) {
		clock := history.(map[string]interface{})["clock"].(string)
		value := history.(map[string]interface{})["value"].(string)
		value_float64, err := strconv.ParseFloat(value, 64)
		if err != nil {
			panic(err)
		}
		score := cp.Update(value_float64)
		int64_clock, _ := strconv.ParseInt(clock, 10, 64)
		if int64_clock > time.Now().Unix()-interval {
			send_data = append(send_data, zabbix_sender.DataItem{Hostname: host_name, Key: item_prefix + "." + orig_item_key, Value: strconv.FormatFloat(score, 'f', 10, 64), Timestamp: int64_clock})
		}
		fmt.Printf("%s\t%s\t%f\n", clock, value, score)
	}
	addr, _ := net.ResolveTCPAddr("tcp", zabbix_host+":"+zabbix_port)
	if len(send_data) > 0 {
		res, err := zabbix_sender.Send(addr, send_data)
		if err != nil {
			fmt.Printf("[ERROR]: zabbix sender error!: %s", err)
			os.Exit(1)
		}
		fmt.Printf("[INFO]: Successful sending data to Zabbix: %s", res)
	} else {
		fmt.Printf("[INFO]: No sending data")
	}
}
