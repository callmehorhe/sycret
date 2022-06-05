package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"log"

	"github.com/gin-gonic/gin"
)

type Service struct {
	File     []byte
	RecordId string
}

func Start(c *gin.Context) {
	fileUrl := c.Request.URL.Query().Get("URLTemplate")
	recid := c.Request.URL.Query().Get("RecordID")
	file, err := getFile(fileUrl)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "wrong url",
		})
	}
	service := newService(file, recid)
}

func newService(file []byte, recordId string) *Service {
	return &Service{
		File:     file,
		RecordId: recordId,
	}
}

func (s *Service) findAndReplace(field, newData string) {
	r := regexp.MustCompile(fmt.Sprintf("(?s)<ns1:text field=\"%s\">(.*?)</ns1:text>", field))
	text := r.Find(s.File)
	textreg := regexp.MustCompile("(?s)<w:t>(.*?)</w:t>")
	newtext := textreg.ReplaceAll(text, []byte(fmt.Sprintf("<w:t> %s </w:t>", newData)))
	s.File = r.ReplaceAll(s.File, newtext)
}

func Startt(c *gin.Context) {
	file := c.Request.URL.Query().Get("URLTemplate")
	recid := c.Request.URL.Query().Get("RecordID")

	f, err := getFile(file)
	if err != nil {
		log.Fatal(err)
	}

	findAndReplace := func(field, newData string) {
		r := regexp.MustCompile(fmt.Sprintf("(?s)<ns1:text field=\"%s\">(.*?)</ns1:text>", field))
		text := r.Find(f)
		textreg := regexp.MustCompile("(?s)<w:t>(.*?)</w:t>")
		newtext := textreg.ReplaceAll(text, []byte(fmt.Sprintf("<w:t> %s </w:t>", newData)))
		f = r.ReplaceAll(f, newtext)
	}

	textFields := findTags(f)

	for _, field := range textFields {
		data, err := getData(field, recid)
		if data.Description != "OK" || err != nil {
			log.Print("Wrong input")
			return
		}

		d := strings.Split(data.ResultData, " ")[0]
		for _, field := range textFields {
			findAndReplace(field, d)
		}
	}

	now := time.Now()

	n, err := os.Create(fmt.Sprintf("%04d-%02d-%02d %02d-%02d-%02d.doc", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second()))
	if err != nil {
		log.Fatal(err)
	}
	n.Write(f)
}

func getFile(url string) ([]byte, error) {
	c := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal("error: ", err)
		return []byte{}, err
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/102.0.5005.63 Safari/537.36")
	res, err := c.Do(req)
	if err != nil {
		log.Fatal("error: ", err)
		return []byte{}, err
	}

	return ioutil.ReadAll(res.Body)
}

type Data struct {
	Result      int    `json:"result"`
	Description string `json:"resultdescription"`
	ResultData  string `json:"resultdata"`
}

func getData(s string, id string) (Data, error) {
	c := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://sycret.ru/service/apigendoc/apigendoc?text=%s&recordid=%s", s, id), nil)
	if err != nil {
		log.Fatal("error: ", err)
		return Data{}, err
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/102.0.5005.63 Safari/537.36")
	res, err := c.Do(req)
	if err != nil {
		log.Fatal("error: ", err)
		return Data{}, err
	}
	var data Data
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		log.Fatal(err)
		return Data{}, err
	}

	return data, nil
}

//Find all values of attr "field"
func findTags(file []byte) []string {
	var textFields []string

	decoder := xml.NewDecoder(bytes.NewReader(file))
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		switch v := token.(type) {
		case xml.StartElement:
			if v.Name.Local == "text" {
				for _, a := range v.Attr {
					if a.Name.Local == "field" {
						textFields = append(textFields, a.Value)
					}
				}
			}
		}
	}
	return textFields
}
