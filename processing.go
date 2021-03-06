package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-resty/resty"
	"github.com/gorilla/feeds"
	"github.com/pkg/errors"
)

func getGroupDataFromVK() (title string, description string, image string, err error) {

	token, domain, proxy := os.Getenv("VKRSS_ACCESS_TOKEN"),
		os.Getenv("VKRSS_DOMAIN"),
		os.Getenv("VKRSS_PROXY")

	// Make query
	params := make(map[string]string)

	params["access_token"] = token
	params["group_ids"] = domain
	params["fields"] = "description"

	params["lang"] = languageVK
	params["v"] = apiVersion

	client := resty.New()

	if proxy != "" {
		client = client.SetProxy(proxy)
	}

	resp, err := client.R().
		SetQueryParams(params).
		Get(vkURL + "groups.getById")

	if err != nil {
		log.Fatalln("[Error] VK::CallMethod:", err.Error(), "WebResponse:", string(resp.Body()))
		return "", "", "", err
	}

	var body GroupData

	if err := json.Unmarshal(resp.Body(), &body); err != nil {
		log.Fatalln("[Error] VK::CallMethod:", err.Error(), "WebResponse:", string(resp.Body()))
	}

	if body.Error != nil {
		if errorMsg, exists := body.Error["error_msg"].(string); exists {
			log.Fatalln("[Error] VK::CallMethod:", errorMsg, "WebResponse:", string(resp.Body()))
			return "", "", "", errors.New(errorMsg)
		}

		log.Fatalln("[Error] VK::CallMethod:", "Unknown error", "WebResponse:", string(resp.Body()))

		return "", "", "", errors.New(resp.String())
	}

	title = body.Response[0].Name
	description = body.Response[0].Description

	if body.Response[0].Photo200 != "" {
		image = body.Response[0].Photo200
	} else if body.Response[0].Photo100 != "" {
		image = body.Response[0].Photo100
	} else if body.Response[0].Photo50 != "" {
		image = body.Response[0].Photo50
	}

	return title, description, image, nil
}

func getDataFromVK() JSONBody {

	token, domain, filter, count, proxy := os.Getenv("VKRSS_ACCESS_TOKEN"),
		os.Getenv("VKRSS_DOMAIN"),
		os.Getenv("VKRSS_FILTER"),
		os.Getenv("VKRSS_COUNT"),
		os.Getenv("VKRSS_PROXY")

	// Make query
	params := make(map[string]string)

	params["access_token"] = token
	params["domain"] = domain
	params["filter"] = filter
	params["count"] = count

	params["lang"] = languageVK
	params["v"] = apiVersion

	client := resty.New()

	if proxy != "" {
		client = client.SetProxy(proxy)
	}

	resp, err := client.R().
		SetQueryParams(params).
		Get(vkURL + "wall.get")

	if err != nil {
		log.Fatalln("[Error] VK::CallMethod:", err.Error(), "WebResponse:", string(resp.Body()))
	}

	var body JSONBody

	if err := json.Unmarshal(resp.Body(), &body); err != nil {
		log.Fatalln("[Error] VK::CallMethod:", err.Error(), "WebResponse:", string(resp.Body()))
	}

	if body.Error != nil {
		if errorMsg, exists := body.Error["error_msg"].(string); exists {
			log.Fatalln("[Error] VK::CallMethod:", errorMsg, "WebResponse:", string(resp.Body()))
		}

		log.Fatalln("[Error] VK::CallMethod:", "Unknown error", "WebResponse:", string(resp.Body()))
	}

	return body
}

func dataToRSS(data JSONBody) (string, error) {

	now := time.Unix(int64(data.Response.Items[0].Date), 0)

	title, description, image, err := getGroupDataFromVK()

	if err != nil {
		title = "error"
		description = "vk wrong answer: " + err.Error()
		log.Fatalln("[Error] VK::GroupData:", "Unknown error", "Err:", err)
	}

	feed := &feeds.Feed{
		Title:       title,
		Link:        &feeds.Link{Href: "https://vk.com/" + os.Getenv("VKRSS_DOMAIN")},
		Description: description,
		Updated:     now,
		Created:     now,
		Copyright:   "riftbit.com",
		Image: &feeds.Image{
			Url:   image,
			Title: title,
			Link:  "https://vk.com/" + os.Getenv("VKRSS_DOMAIN"),
		},
	}

	for _, element := range data.Response.Items {

		var concreteData Item

		if element.CopyHistory != nil {
			concreteData = element.CopyHistory[0]
		} else {
			concreteData = element
		}

		preparedContent := strings.Replace(concreteData.Text, "\n", "<br>", -1) + "<br>"

		if os.Getenv("VKRSS_NEED_IMAGES") != "" {
			for _, photo := range concreteData.Attachments {
				if photo.Type == "photo" {

					photoLink := ""

					if photo.Photo.Photo1280 != "" {
						photoLink = photo.Photo.Photo1280
					} else if photo.Photo.Photo807 != "" {
						photoLink = photo.Photo.Photo807
					} else if photo.Photo.Photo604 != "" {
						photoLink = photo.Photo.Photo604
					} else if photo.Photo.Photo130 != "" {
						photoLink = photo.Photo.Photo130
					}

					if photoLink != "" {
						preparedContent += "<img src='" + photoLink + "'><br>"
					}
				}
			}
		}

		if os.Getenv("VKRSS_NEED_AUDIOS") != "" {
			for _, audio := range concreteData.Attachments {
				if audio.Type == "audio" {
					preparedContent += fmt.Sprintf("🎧 %s<br>", audio.Audio.Artist+" - "+audio.Audio.Title)
				}
			}
		}

		if os.Getenv("VKRSS_NEED_VIDEOS") != "" {
			for _, video := range concreteData.Attachments {
				if video.Type == "video" {

					videoImage := ""
					if video.Video.Photo800 != "" {
						videoImage = video.Video.Photo800
					} else if video.Video.Photo640 != "" {
						videoImage = video.Video.Photo640
					} else if video.Video.Photo320 != "" {
						videoImage = video.Video.Photo320
					} else if video.Video.Photo130 != "" {
						videoImage = video.Video.Photo130
					}

					preparedContent += fmt.Sprintf("🎬 %s<br><img src='%s'><br>", video.Video.Title, videoImage)
				}
			}
		}

		if os.Getenv("VKRSS_NEED_DOCS") != "" {
			for _, doc := range concreteData.Attachments {
				if doc.Type == "doc" {
					preparedContent += fmt.Sprintf("💾 <a href='%s'>%s</a><br>", doc.Doc.URL, doc.Doc.Title)
				}
			}
		}

		item := &feeds.Item{
			Title:       "",
			Link:        &feeds.Link{Href: fmt.Sprintf("https://vk.com/wall%d_%d", element.OwnerID, element.ID)},
			Source:      nil,
			Author:      nil,
			Description: "",
			Id:          fmt.Sprintf("%d_%d", element.OwnerID, element.ID),
			Updated:     time.Unix(int64(element.Date), 0),
			Created:     time.Unix(int64(element.Date), 0),
			Enclosure:   nil,
			Content:     preparedContent,
		}

		feed.Add(item)

	}

	return feed.ToRss()

}
