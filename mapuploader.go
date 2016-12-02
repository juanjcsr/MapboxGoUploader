package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	"path"

	"net/http"

	"time"

	"gopkg.in/urfave/cli.v1"
)

func main() {

	mapboxHTTPClient := &http.Client{
		Timeout: time.Second * 30,
	}

	app := cli.NewApp()
	app.Name = "mapuploader"
	app.Usage = "uploads data to mapbox servers"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "mapbox_key, mk",
			Usage:  "the key for mapbox api access",
			EnvVar: "MAPBOX_KEY",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:    "upload",
			Aliases: []string{"u"},
			Usage:   "uploads the geojson or tile to mapbox",
			Action: func(c *cli.Context) error {
				fmt.Println("uploading file: ", c.Args().First())
				getMapboxCredentials("holahola", "juanjcsr", mapboxHTTPClient)
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

type mapboxCredentials struct {
	AccessKeyID     string
	Bucket          string
	Key             string
	SecretAccessKey string
	SessionToken    string
	URL             string
}

func getMapboxCredentials(apiKey string, userName string, mapboxClient *http.Client) {
	baseURL, _ := url.Parse(MapboxURL)
	uploadsURLSegment := fmt.Sprintf("/uploads/v1/%s/credentials", userName)
	parameters := url.Values{}
	parameters.Add("access_token", apiKey)
	baseURL.Path = path.Join(baseURL.Path, uploadsURLSegment)
	baseURL.RawQuery = parameters.Encode()
	fmt.Println(baseURL.String())

	response, err := mapboxClient.Get(baseURL.String())
	defer response.Body.Close()
	if err != nil {
		credentials := new(mapboxCredentials)
		if err := json.NewDecoder(response.Body).Decode(credentials); err != nil {
			fmt.Println(credentials)
		}
	}

}
