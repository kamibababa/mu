package main

import (
	"crawler/lib"
	"crawler/util"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Tag struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type Tab struct {
	Name string `json:"name"`
	Key  string `json:"key"`
	Tags []Tag  `json:"tags"`
}

func JSON(w http.ResponseWriter, data []byte) {
	w.Header().Set("content-type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(data)
}

func config(w http.ResponseWriter, req *http.Request) {
	var tabs []Tab

	var fetchTags = func(tabs []map[string]string) []Tag {
		var tags []Tag
		for _, v := range tabs {
			tags = append(tags, Tag{
				Name: v["name"],
				Key:  v["tag"],
			})
		}

		return tags
	}

	sites := []string{
		lib.SITE_V2EX,
		lib.SITE_CT,
		lib.SITE_ZHIHU,
		lib.SITE_WEIBO,
		lib.SITE_HACKER,
	}

	for _, s := range sites {
		st := lib.NewSite(s)
		tabs = append(tabs, Tab{
			Name: st.Name,
			Key:  st.Key,
			Tags: fetchTags(st.Tabs),
		})
	}

	data, _ := json.Marshal(tabs)

	JSON(w, data)
}

func aj(w http.ResponseWriter, req *http.Request) {
	client := lib.RedisConn()
	defer client.Close()

	key := req.URL.Query()["key"][0]
	hkey := req.URL.Query()["hkey"][0]

	data, err := client.HGet(key, hkey).Result()

	if err != nil {
		log.Println("[info] aj req empty " + err.Error())
		JSON(w, []byte(`{"list": [], "t":""}`))
		return
	}

	var hotJson lib.HotJson
	err = json.Unmarshal([]byte(data), &hotJson)
	if err != nil {
		log.Println("[error] aj req error " + err.Error())
		JSON(w, []byte(`{"list": [], "t":""}`))
		return
	}

	js, _ := json.Marshal(hotJson)

	JSON(w, []byte(js))
}

func welcome() {
	fmt.Println(` __  __ _   _`)
	fmt.Println(`|  \/  | | | |`)
	fmt.Println(`| |\/| | | | |`)
	fmt.Println(`| |  | | |_| |`)
	fmt.Println(`|_|  |_|\___/`)
	fmt.Println("welcome ~")
}

func main() {
	appConfig := util.NewConfig()

	fs := http.FileServer(http.Dir("public"))
	http.Handle("/", fs)
	http.HandleFunc("/aj", aj)
	http.HandleFunc("/config", config)

	welcome()
	log.Println("listen on " + appConfig.Addr)

	log.Fatal(http.ListenAndServe(appConfig.Addr, nil))
}
