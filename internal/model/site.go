package model

import (
	"crawler/internal/svc/lib"
	"crawler/internal/util/logger"
	"encoding/json"
	"errors"
	"strings"
)

type CrawType int
type NodeOption int8
type Status int8

const (
	CrawHtml CrawType = 1 // 网站是HTML
	CrawApi  CrawType = 2 // 网站是JSON接口

	ByType  NodeOption = 1 // 通过服务器类型
	ByHosts NodeOption = 2 // 服务器IPs

	Disable Status = 0 // 禁用
	Enable  Status = 1 // 启用
)

type Site struct {
	ID         int
	Name       string     `gorm:"name"`
	Root       string     `gorm:"root"`
	Key        string     `gorm:"key"`
	Desc       string     `gorm:"desc"`
	Type       int8       `gorm:"type"`
	Tags       string     `gorm:"tags"`
	Cron       string     `gorm:"cron"`
	Enable     Status     `gorm:"enable"`
	NodeOption NodeOption `gorm:"node_option"`
	NodeType   int8       `gorm:"node_type"`
	NodeHosts  string     `gorm:"node_hosts"`
}

type Tag struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	Enable int8   `json:"enable"`
}

type SiteJson struct {
	ID         int        `json:"id"`
	Name       string     `json:"name"`
	Key        string     `json:"key"`
	Root       string     `json:"root"`
	Desc       string     `json:"desc"`
	Tags       []Tag      `json:"tags"`
	Type       int8       `json:"type"`
	Cron       string     `json:"cron"`
	NodeOption NodeOption `json:"node_option"`
	NodeType   int8       `json:"node_type"`
	NodeHosts  []int      `json:"node_hosts"`
	Enable     Status     `json:"enable"`
}

func (s *Site) TableName() string {
	return "site"
}

func (s *Site) CheckArgs() error {
	if s.Name == "" {
		return errors.New("名字为空")
	}
	if len(strings.Split(s.Cron, " ")) != 5 {
		return errors.New("cron必须是5位表达式")
	}

	return nil
}

func (s *Site) Create() error {
	tmp, err := s.FetchRows("`key` = ? or `root` = ?", s.Key, s.Root)
	if err != nil {
		return errors.New("create site error " + err.Error())
	}
	if len(tmp) > 0 {
		return errors.New("同key或者同root的站点已经存在")
	}

	db := DPool().Conn
	defer db.Close()

	db = db.Create(&s)
	if err = db.Error; err != nil {
		logger.Error("create err %v, exp %s .", err, db.QueryExpr())
		return errors.New("create site err")
	}

	return nil
}

func (s *Site) Update(data map[string]interface{}) error {
	db := DPool().Conn
	defer db.Close()

	db = db.Model(&s).Update(data)
	if err := db.Error; err != nil {
		logger.Error("update err %v, exp %s .", err, db.QueryExpr())
		return errors.New("update site failed")
	}

	return nil
}

func (s *Site) FetchInfo() (Site, error) {
	var tmp Site
	db := DPool().Conn
	defer db.Close()

	db = db.Where("id = ?", s.ID).First(&tmp)
	if err := db.Error; err != nil && !db.RecordNotFound() {
		logger.Error("FetchInfo err %v, exp %s .", err, db.QueryExpr())
		return Site{}, errors.New("fetch site info failed")
	}

	return tmp, nil
}

func (s *Site) FetchRow(query string, args ...interface{}) (Site, error) {
	db := DPool().Conn
	defer db.Close()

	var site Site
	db = db.Where(query, args...).First(&site)
	if err := db.Error; err != nil && !db.RecordNotFound() {
		logger.Error("FetchRows err %v, exp %s .", err, db.QueryExpr())
		return Site{}, errors.New("fetchRow site failed")
	}
	return site, nil
}

func (s *Site) FetchRows(query string, args ...interface{}) ([]Site, error) {
	db := DPool().Conn
	defer db.Close()

	var list []Site
	db = db.Where(query, args...).Find(&list)
	if err := db.Error; err != nil {
		logger.Error("FetchRows err %v, exp %s .", err, db.QueryExpr())
		return nil, errors.New("fetchRows site failed")
	}
	return list, nil
}

func (s *Site) FormatJson() (SiteJson, error) {
	var tags []Tag
	var hosts []int

	var err error
	if s.Tags != "" {
		err = json.Unmarshal([]byte(s.Tags), &tags)
		if err != nil {
			return SiteJson{}, errors.New("标签解析失败")
		}
	}

	if s.NodeHosts != "" {
		err = json.Unmarshal([]byte(s.NodeHosts), &hosts)
		if err != nil {
			return SiteJson{}, errors.New("节点解析失败")
		}
	} else {
		hosts = []int{}
	}

	return SiteJson{
		ID:         s.ID,
		Name:       s.Name,
		Key:        s.Key,
		Root:       s.Root,
		Desc:       s.Desc,
		Tags:       tags,
		Type:       s.Type,
		Cron:       s.Cron,
		NodeOption: s.NodeOption,
		NodeType:   s.NodeType,
		NodeHosts:  hosts,
		Enable:     s.Enable,
	}, nil
}

func (s *Site) InitSites() {
	var tagStr []byte

	avaSites := lib.AvailableSites()
	for _, siteKey := range avaSites {
		site := lib.NewSite(siteKey)
		row, err := s.FetchRow(" `key` = ? ", site.Key)
		if err != nil {
			panic("init sites fetch failed " + err.Error())
		}

		var tags []Tag
		for _, tag := range site.Tabs {
			tags = append(tags, Tag{
				Key:    tag["tag"],
				Name:   tag["name"],
				Enable: 1,
			})
		}
		tagStr, _ = json.Marshal(tags)

		if row.ID > 0 {
			err = row.Update(map[string]interface{}{
				"name": site.Name,
				"root": site.Root,
				"tags": string(tagStr),
				"type": site.CrawType,
			})
			if err != nil {
				panic("init sites update failed " + err.Error())
			}
			continue
		}

		row = Site{
			Name:       site.Name,
			Key:        site.Key,
			Root:       site.Root,
			Cron:       "*/30 * * * *",
			NodeOption: 1, // 默认使用分类
			NodeType:   1, // 默认国内的机器
			NodeHosts:  "",
			Desc:       site.Desc,
			Tags:       string(tagStr),
			Type:       site.CrawType,
		}
		err = row.Create()
		if err != nil {
			panic("init sites create failed " + err.Error())
		}
	}
}

func (s *Site) FixNodeId(delId int) {
	sites, _ := (&Site{}).FetchRows("1 = 1 ")
	for _, site := range sites {
		sj, _ := site.FormatJson()
		newHosts := []int{}
		for _, v := range sj.NodeHosts {
			if v == delId {
				continue
			}
			newHosts = append(newHosts, v)
		}
		jstr, _ := json.Marshal(newHosts)
		_ = site.Update(map[string]interface{}{
			"node_hosts": jstr,
		})
	}
}