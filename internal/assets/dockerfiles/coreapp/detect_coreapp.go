// +build ignore

/*
Copyright IBM Corporation 2020

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	csProj      = ".csproj"
	DotNETCore5 = "net5.0"
)

type ConfigInfo struct {
	version   string
	ports     []int
	httpPort  int
	httpsPort int
	appName   string
	path      string
}

type LaunchSettings struct {
	Profiles map[string]interface{} `json:"profiles"`
}

type ConfigurationDotNETCore struct {
	XMLName       xml.Name      `xml:"Project"`
	Sdk           string        `xml:"Sdk,attr"`
	PropertyGroup PropertyGroup `xml:"PropertyGroup"`
}

type PropertyGroup struct {
	XMLName         xml.Name `xml:"PropertyGroup"`
	Condition       string   `xml:"Condition,attr"`
	TargetFramework string   `xml:"TargetFramework"`
}

//GetFilesByExt returns files by extension
func GetFilesByExt(inputPath string, exts []string) ([]string, error) {
	var files []string
	if info, err := os.Stat(inputPath); os.IsNotExist(err) {
		log.Warnf("Error in walking through files due to : %q", err)
		return nil, err
	} else if !info.IsDir() {
		log.Warnf("The path %q is not a directory.", inputPath)
	}
	err := filepath.Walk(inputPath, func(path string, info os.FileInfo, err error) error {
		if err != nil && path == inputPath { // if walk for root search path return gets error
			// then stop walking and return this error
			return err
		}
		if err != nil {
			log.Warnf("Skipping path %q due to error: %q", path, err)
			return nil
		}
		// Skip directories
		if info.IsDir() {
			return nil
		}
		fext := filepath.Ext(path)
		for _, ext := range exts {
			if fext == ext {
				files = append(files, path)
			}
		}
		return nil
	})
	if err != nil {
		log.Warnf("Error in walking through files due to : %q", err)
		return files, err
	}
	log.Debugf("No of files with %s ext identified : %d", exts, len(files))
	return files, nil
}

//DetectDotNETCore returns .NET version if detects .NETCore
func DetectDotNETCore(sourcePath string) (bool, ConfigInfo) {
	var configInfo ConfigInfo
	csprojFiles, err := GetFilesByExt(sourcePath, []string{csProj})
	if err != nil {
		fmt.Println(err)
	}
	for _, csprojFile := range csprojFiles {
		xmlFile, err := os.Open(csprojFile)
		if err != nil {
			fmt.Println(err)
		}

		defer xmlFile.Close()

		byteValue, _ := ioutil.ReadAll(xmlFile)

		configuration := ConfigurationDotNETCore{}
		xml.Unmarshal(byteValue, &configuration)
		configInfo.version = configuration.PropertyGroup.TargetFramework
		configInfo.path, err = filepath.Rel(sourcePath, csprojFile)
		if err != nil {
			fmt.Println(err)
		}
		configInfo.appName = strings.TrimSuffix(filepath.Base(csprojFile), filepath.Ext(csprojFile))
		// fmt.Printf("App name %v\n", configInfo.appName)
	}
	// fmt.Printf("%v\n", configInfo.version)
	if configInfo.version == DotNETCore5 {
		jsonFiles, err := GetFilesByExt(sourcePath, []string{".json"})
		if err != nil {
			fmt.Println(err)
		}
		for _, jsonFile := range jsonFiles {
			if filepath.Base(jsonFile) == "launchSettings.json" {
				settingsJsonFile, err := os.Open(jsonFile)
				if err != nil {
					fmt.Println(err)
				}

				defer settingsJsonFile.Close()

				byteValue, _ := ioutil.ReadAll(settingsJsonFile)
				// fmt.Printf("byteValue %v\n", byteValue)
				launchSettings := LaunchSettings{}
				json.Unmarshal(byteValue, &launchSettings)
				// fmt.Printf("launchSettings %v\n", launchSettings)

				if launchSettings.Profiles[configInfo.appName] != nil {
					profiles := launchSettings.Profiles[configInfo.appName].(map[string]interface{})
					applicationUrls := profiles["applicationUrl"].(string)
					Urls := strings.Split(applicationUrls, ";")
					re := regexp.MustCompile("[0-9]+")

					for _, url := range Urls {
						re1 := regexp.MustCompile("^https://")
						if len(re1.FindAllString(url, -1)) != 0 && re1.FindAllString(url, -1)[0] == "https://" {
							port, _ := strconv.Atoi(re.FindAllString(url, 1)[0])
							configInfo.httpsPort = port
						}

						re2 := regexp.MustCompile("^http://")
						if len(re2.FindAllString(url, -1)) != 0 && re2.FindAllString(url, -1)[0] == "http://" {
							port, _ := strconv.Atoi(re.FindAllString(url, 1)[0])
							configInfo.httpPort = port
						}
					}
					configInfo.ports = append(configInfo.ports, configInfo.httpPort)
					// configInfo.ports = append(configInfo.ports, configInfo.httpsPort)
					// fmt.Printf("Port %v\n", configInfo.ports)
				}
			}
		}

		return true, configInfo
	}

	return false, configInfo
}

func main() {

	if len(os.Args) < 2 {
		log.Warnf("Source path is missing in the argument")
	} else {
		isDotNETCore, configInfo := DetectDotNETCore(os.Args[1])
		if isDotNETCore {
			// fmt.Printf("{\"csprojPath\":  \"%s\", \"httpPort\": %v, \"httpsPort\": %v, \"appName\": \"%s\"}\n", configInfo.path, configInfo.httpPort, configInfo.httpsPort, configInfo.appName)
			fmt.Printf("{\"csprojPath\":  \"%s\", \"ports\": [", configInfo.path)
			for i, port := range configInfo.ports {
				fmt.Printf("%d", port)
				if i != len(configInfo.ports)-1 {
					fmt.Printf(",")
				}
			}
			fmt.Printf("], \"httpPort\": %v, \"appName\": \"%s\"}\n", configInfo.httpPort, configInfo.appName)
		} else {
			os.Exit(1)
		}
	}
}
