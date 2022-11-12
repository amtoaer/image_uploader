package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
)

type Uploader struct {
	URL          string
	ResultGetter string
	Header       map[string][]string
}

type Config struct {
	Active   string
	Uploader map[string]Uploader
}

type Images struct {
	Path     []string
	uploader Uploader
}

func NewImages(path []string) *Images {
	var validPath []string
	for _, imagePath := range path {
		if _, err := os.Stat(imagePath); err != nil {
			fmt.Printf("get %s stat error, skiping..\n", imagePath)
			continue
		}
		validPath = append(validPath, imagePath)
	}
	return &Images{
		Path: validPath,
	}
}

func (i *Images) WithUploader(uploader Uploader) *Images {
	i.uploader = uploader
	return i
}

func (i *Images) Upload() {
	fmt.Printf("uploading %s...\n", i.Path)
	client := http.Client{}
	var uploadURL []string
	for _, imagePath := range i.Path {
		buf := new(bytes.Buffer)
		mpw := multipart.NewWriter(buf)
		part, _ := mpw.CreateFormFile("file", filepath.Base(imagePath))
		content, _ := os.ReadFile(imagePath)
		part.Write(content)
		mpw.Close()
		req, _ := http.NewRequest("POST", i.uploader.URL, buf)
		req.Header = i.uploader.Header
		req.Header.Set("Content-Type", mpw.FormDataContentType())
		res, err := client.Do(req)
		if err != nil {
			fmt.Printf("upload %s error: failed to request\n", imagePath)
			continue
		}
		body, err := io.ReadAll(res.Body)
		defer res.Body.Close()
		if err != nil {
			fmt.Printf("upload %s error: failed to read response body\n", imagePath)
			continue
		}
		var result map[string]any
		err = json.Unmarshal(body, &result)
		if err != nil {
			fmt.Printf("upload %s error: failed to parse response body\n", imagePath)
			continue
		}
		var (
			ok  bool = true
			url string
		)
		getters := strings.Split(i.uploader.ResultGetter, ".")
		for idx, getter := range strings.Split(i.uploader.ResultGetter, ".") {
			if idx == len(getters)-1 {
				url, ok = result[getter].(string)
				if !ok {
					break
				}
			} else {
				result, ok = result[getter].(map[string]any)
				if !ok {
					break
				}
			}
		}
		if !ok {
			fmt.Printf("upload %s error: failed to get upload url\n", imagePath)
			uploadURL = append(uploadURL, "")
		} else {
			uploadURL = append(uploadURL, url)
		}
	}
	fmt.Println(strings.Join(uploadURL, "\n"))
}

func handleError(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func tryInitConfigFile() error {
	configPath, err := homedir.Expand("~/.iu")
	if err != nil {
		return fmt.Errorf("can't get homedir")
	}
	if _, err := os.Stat(configPath); err != nil {
		if !os.IsExist(err) {
			emptyConfig, _ := json.MarshalIndent(Config{
				Active: "dogedoge",
				Uploader: map[string]Uploader{
					"dogedoge": {
						URL:          "",
						ResultGetter: "",
						Header:       map[string][]string{},
					},
				},
			}, "", "    ")
			if err = os.WriteFile(configPath, emptyConfig, 0644); err != nil {
				return fmt.Errorf("init config file failed")
			}
		} else {
			return fmt.Errorf("can't get config file info")
		}
	}
	return nil
}

func readConfig() (Config, error) {
	conf := Config{}
	configPath, err := homedir.Expand("~/.iu")
	if err != nil {
		return conf, fmt.Errorf("can't get homedir")
	}
	if content, err := os.ReadFile(configPath); err != nil {
		return conf, fmt.Errorf("read config failed")
	} else {
		if err = json.Unmarshal(content, &conf); err != nil {
			return conf, fmt.Errorf("can't parse config file")
		}
		return conf, nil
	}
}

func main() {
	err := tryInitConfigFile()
	handleError(err)
	conf, err := readConfig()
	handleError(err)
	if uploader, ok := conf.Uploader[conf.Active]; !ok {
		fmt.Printf("get uploader error: unknown uploader %s\n", conf.Active)
	} else {
		NewImages(os.Args[1:]).WithUploader(uploader).Upload()
	}
}
