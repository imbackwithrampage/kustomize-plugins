package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"gopkg.in/yaml.v3"
)

func readYamlFile(path string, dest interface{}) {
	configFile, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	configDecoder := yaml.NewDecoder(configFile)
	if err := configDecoder.Decode(dest); err != nil {
		panic(err)
	}
}

func readFile(config *SourceConfig) ([]byte, error) {
	var path string
	switch p := config.Args["path"].(type) {
	case string:
		path = p
	default:
		panic(errors.New("path missing from file type source"))
	}

	u, err := url.Parse(path)

	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "s3":
		awsRoleArn := getString(config.Args, "awsRoleArn")
		awsRegion := getString(config.Args, "awsRegion")

		sess := session.Must(session.NewSession())
		creds := sess.Config.Credentials

		if awsRoleArn != "" {
			creds = stscreds.NewCredentials(sess, awsRoleArn)
		}

		var region *string = nil
		if awsRegion != "" {
			region = &awsRegion
		}

		bucket := s3.New(sess, &aws.Config{Credentials: creds, Region: region})
		goi := s3.GetObjectInput{}
		goi.Bucket = &u.Host
		goi.Key = &u.Path

		goo, err := bucket.GetObject(&goi)
		if err != nil {
			return nil, err
		}

		data, err := ioutil.ReadAll(goo.Body)
		if err != nil {
			return nil, err
		}

		return data, nil
	case "":
		return os.ReadFile(path)
	default:
		return nil, errors.New("unsupported URL scheme: " + u.Scheme)
	}
}

func convertFileConfig(config *SourceConfig) map[string]string {
	data, err := readFile(config)
	if err != nil {
		panic(err)
	}

	path := getString(config.Args, "path")
	var raw map[string]interface{}
	switch {
	case strings.HasSuffix(path, ".yml"), strings.HasSuffix(path, ".yaml"):
		if err := yaml.Unmarshal(data, &raw); err != nil {
			panic(err)
		}
	case strings.HasSuffix(path, ".json"):
		if err := json.Unmarshal(data, &raw); err != nil {
			panic(err)
		}
	default:
		panic(errors.New("unsupported variable file type"))
	}

	flat := make(map[string]string)
	flattenToMap(raw, "", flat)
	return flat
}
