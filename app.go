package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
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
		return
	}
	s := newService(file, recid)
	textFields := s.findTags()
	if len(textFields) < 1 {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "no such attributes",
		})
		return
	}
	err = s.modifyDocumnet(textFields)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "wrong data",
		})
		return
	}
	url, err := s.createModifiedFile()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	fmt.Println(url)
	c.Status(http.StatusOK)
	encode := json.NewEncoder(c.Writer)
	encode.SetEscapeHTML(false)
	encode.Encode(gin.H{
		"URLWorld": url,
	})
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

func newService(file []byte, recordId string) *Service {
	return &Service{
		File:     file,
		RecordId: recordId,
	}
}

//Find all values of attr "field"
func (s *Service) findTags() []string {
	var textFields []string

	decoder := xml.NewDecoder(bytes.NewReader(s.File))
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

func (s *Service) findAndReplace(field, newData string) {
	r := regexp.MustCompile(fmt.Sprintf("(?s)<ns1:text field=\"%s\">(.*?)</ns1:text>", field))
	text := r.Find(s.File)
	textreg := regexp.MustCompile("(?s)<w:t>(.*?)</w:t>")
	newtext := textreg.ReplaceAll(text, []byte(fmt.Sprintf("<w:t> %s </w:t>", newData)))
	s.File = r.ReplaceAll(s.File, newtext)
}

func (s *Service) modifyDocumnet(textFields []string) error {
	for _, field := range textFields {
		data, err := s.getData(field)
		if data.Description != "OK" || err != nil {
			log.Print("Wrong input")
			return errors.New("wrong data")
		}

		s.findAndReplace(field, data.ResultData)
	}
	return nil
}

type Data struct {
	Result      int    `json:"result"`
	Description string `json:"resultdescription"`
	ResultData  string `json:"resultdata"`
}

func (s *Service) getData(field string) (Data, error) {
	c := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://sycret.ru/service/apigendoc/apigendoc?text=%s&recordid=%s", field, s.RecordId), nil)
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

func (s *Service) createModifiedFile() (string, error) {
	now := time.Now()
	fileName := fmt.Sprintf("%04d-%02d-%02d %02d-%02d-%02d.doc", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
	n, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	n.Write(s.File)
	defer n.Close()
	return openFile(fileName)
}

func openFile(fileName string) (string, error) {
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	path := fmt.Sprintf("app%%3A%%2F%s", fileName)
	err = uploadFile(file, path)
	if err != nil {
		return "", err
	}
	href, err := getDownloadLink(strings.ReplaceAll(path, " ", "%20"))
	if err != nil {
		return "", err
	}
	return href, nil
}

var token = "AQAAAABXzuWQAAf1f3Ik9yK3H0r4qJc5KLRLTjY"
var ynxUrl = "https://cloud-api.yandex.net/v1/disk"

func apiRequest(path, method string) (*http.Response, error) {
	client := http.Client{}
	url := fmt.Sprintf("%s/%s", ynxUrl, path)
	req, _ := http.NewRequest(method, url, nil)
	req.Header.Add("Authorization", fmt.Sprintf("OAuth %s", token))
	req.Header.Add("Accept", "application/json")
	return client.Do(req)
}

func uploadFile(data *os.File, remotePath string) error {
	// функция получения url для загрузки файла
	getUploadUrl := func(path string) (string, error) {
		res, err := apiRequest(fmt.Sprintf("resources/upload?path=%s&overwrite=true", path), "GET")
		if err != nil {
			return "", err
		}
		var resultJson struct {
			Href string `json:"href"`
		}
		err = json.NewDecoder(res.Body).Decode(&resultJson)
		if err != nil {
			return "", err
		}
		return resultJson.Href, err
	}

	// получем ссылку для загрузки файла
	href, err := getUploadUrl(remotePath)
	if err != nil {
		return err
	}

	// загружаем файл по полученной ссылке методом PUT
	req, err := http.NewRequest("PUT", href, data)
	if err != nil {
		return err
	}
	// в header запроса добавляем токен
	req.Header.Add("Authorization", fmt.Sprintf("OAuth %s", token))

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return nil
}

func getDownloadLink(path string) (string, error) {
	res, err := apiRequest(fmt.Sprintf("resources/download?path=%s", path), "GET")
	if err != nil {
		return "", err
	}
	var resultJson struct {
		Href string `json:"href"`
	}
	err = json.NewDecoder(res.Body).Decode(&resultJson)
	if err != nil {
		return "", err
	}
	return resultJson.Href, err
}
