package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"

	"bytes"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"gopkg.in/urfave/cli.v1"
)

func main() {

	mapboxHTTPClient := &http.Client{
		Timeout: time.Second * 30,
	}

	var mapboxAPIKey string
	var mapboxUserName string
	// var mapboxCredentials MapboxCredentials

	app := cli.NewApp()
	app.Name = "mapuploader"
	app.Usage = "uploads data to mapbox servers"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "mapbox_key, mk",
			Usage:       "the key for mapbox api access",
			EnvVar:      "MAPBOX_KEY",
			Destination: &mapboxAPIKey,
		},
		cli.StringFlag{
			Name:        "mapbox_user, u",
			Usage:       "the mapbox username",
			Destination: &mapboxUserName,
		},
	}

	app.Commands = []cli.Command{
		{
			Name:    "upload",
			Aliases: []string{"u"},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "name, n",
					Usage: "the name for the tileset",
				},
			},
			Usage: "uploads the geojson or tile to mapbox",
			Action: func(c *cli.Context) error {
				filestring := c.Args().First()
				if mapboxAPIKey == "" || mapboxUserName == "" {
					return cli.NewExitError("you need to provide mapbox access tokens", 1)
				}
				nameVar := c.String("name")
				if nameVar == "" {
					return cli.NewExitError("you need to provide the name for the tileset", 1)
				}
				mapboxCredentials, err := getMapboxCredentials(mapboxAPIKey, mapboxUserName, mapboxHTTPClient)
				if err != nil {
					return cli.NewExitError("could not retrieve mapbox credentials", 1)
				}
				fmt.Println(mapboxCredentials)
				s3uploader := setAWSVariables(mapboxCredentials.AccessKeyID, mapboxCredentials.SecretAccessKey, mapboxCredentials.SessionToken)
				uploadFile(filestring, s3uploader, mapboxCredentials.Bucket, mapboxCredentials.Key)
				postToMapbox(mapboxCredentials.URL, nameVar, mapboxAPIKey, mapboxUserName, mapboxHTTPClient)
				return nil
			},
		},
	}

	app.Action = func(c *cli.Context) error {
		fmt.Println("hola :D")
		return nil
	}
	app.Run(os.Args)
}

// MapboxURL has the remote mapbox endpoint
const MapboxURL = "https://api.mapbox.com"

// MapboxCredentials holds the keys needed to upload stuff
type MapboxCredentials struct {
	AccessKeyID     string
	Bucket          string
	Key             string
	SecretAccessKey string
	SessionToken    string
	URL             string
}

func getMapboxCredentials(apiKey string, userName string, mapboxClient *http.Client) (*MapboxCredentials, error) {
	baseURL, _ := url.Parse(MapboxURL)
	uploadsURLSegment := fmt.Sprintf("/uploads/v1/%s/credentials", userName)
	parameters := url.Values{}
	parameters.Add("access_token", apiKey)
	baseURL.Path = path.Join(baseURL.Path, uploadsURLSegment)
	baseURL.RawQuery = parameters.Encode()
	fmt.Println(baseURL.String())

	response, err := mapboxClient.Get(baseURL.String())
	defer response.Body.Close()
	if err == nil {
		credentials := new(MapboxCredentials)
		if err := json.NewDecoder(response.Body).Decode(credentials); err == nil {
			fmt.Println(credentials)
			return credentials, nil
		}
		return nil, err
	}
	return nil, err
}

func setAWSVariables(awsAccess string, awsSecret string, awsSession string) *s3manager.Uploader {
	os.Clearenv()
	os.Setenv("AWS_ACCESS_KEY_ID", awsAccess)
	os.Setenv("AWS_SECRET_ACCESS_KEY", awsSecret)
	os.Setenv("AWS_SESSION_TOKEN", awsSession)
	creds := credentials.NewEnvCredentials()
	_, err := creds.Get()
	if err == nil {

		config := aws.NewConfig().WithRegion("us-east-1").WithCredentials(creds)

		// return s3.New(session.New(), config)
		s3Client := s3.New(session.New(), config)
		return s3manager.NewUploaderWithClient(s3Client)
	}
	fmt.Println("Could not get credentials")
	return nil
}

func uploadFile(fileString string, s3uploader *s3manager.Uploader, bucketName string, bucketKey string) {
	file, err := os.Open(fileString)
	if err != nil {
		fmt.Printf("err opening file %s", err)
		return
	}
	defer file.Close()

	fileInfo, _ := file.Stat()
	size := fileInfo.Size()
	buffer := make([]byte, size)
	file.Read(buffer)

	resp, err := s3uploader.Upload(&s3manager.UploadInput{
		Bucket: &bucketName,
		Key:    &bucketKey,
		Body:   file,
	})
	if err == nil {
		fmt.Printf("response %s", awsutil.StringValue(resp))

	} else {
		fmt.Printf("bad response %s", err)
	}

}

func postToMapbox(awsURL string, tileTitle string, apiKey string, userName string, mapboxClient *http.Client) {
	baseURL, _ := url.Parse(MapboxURL)
	uploadsURLSegment := fmt.Sprintf("/uploads/v1/%s", userName)
	parameters := url.Values{}
	parameters.Add("access_token", apiKey)
	baseURL.Path = path.Join(baseURL.Path, uploadsURLSegment)
	baseURL.RawQuery = parameters.Encode()
	awsEncodedURL, _ := url.Parse(awsURL)
	jsonString := fmt.Sprintf(`{
        "url": "%s",
        "tileset": "%s.%s",
		"name": "%s"
        }`, awsEncodedURL.String(), userName, tileTitle, tileTitle)

	fmt.Println("THEJSON: ")
	fmt.Println(jsonString)
	byteStr := []byte(jsonString)
	req, err := http.NewRequest("POST", baseURL.String(), bytes.NewBuffer(byteStr))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := mapboxClient.Do(req)
	defer resp.Body.Close()
	if err != nil {
		fmt.Printf("Bad request %s", err)
	} else {
		fmt.Println("response Status:", resp.Status)
		fmt.Println("response Headers:", resp.Header)
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println("response Body:", string(body))
	}
}
