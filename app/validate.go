package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/justicz/giveanet/common"
)

type FormData struct {
	Message        string
	DisplayName    string
	NumNets        uint64
	ImgKind        string
	ImgData        []byte
	SocialLinkType string
	SocialLink     string
	Private        bool
}

func validateNewMessageForm(r *http.Request) (form FormData, errors []string) {
	const minNets = 1
	const maxNets = 25000
	const maxNameLen = 24
	const minUsernameLen = 1
	const maxUsernameLen = 50
	const maxCustomLinkLen = 200
	const maxMessageLen = 80
	const expectedIconDataLen = 300
	const imgKind = "rgb10x10"

	// Parse POSTed form
	err := r.ParseForm()
	if err != nil {
		errors = append(errors, "Failed to parse form")
		return
	}

	// Does the user want to keep their donation private?
	form.Private = r.PostForm.Get("wantmsg") == ""

	netParseError := "Error parsing net quantity"
	numNetError := fmt.Sprintf("You must select between %d and %d nets", minNets, maxNets)
	quantityType := r.PostForm.Get("netquantitytype")

	// Parse number of nets
	var numNets uint64
	switch quantityType {
	case "reg":
		parsed := strings.Split(r.PostForm.Get("netslider"), ".")[0]
		numNets, err = strconv.ParseUint(parsed, 10, 64)
		if err != nil {
			errors = append(errors, netParseError)
		}
	case "lots":
		parsed := strings.Split(r.PostForm.Get("netnumbox"), ".")[0]
		numNets, err = strconv.ParseUint(parsed, 10, 64)
		if err != nil {
			errors = append(errors, netParseError)
		}
	default:
		errors = append(errors, netParseError)
	}

	// Validate number of nets is reasonable
	if numNets < minNets || numNets > maxNets {
		errors = append(errors, numNetError)
	}

	form.NumNets = numNets

	// Don't parse other fields if private
	if form.Private {
		form.SocialLinkType = common.SocialLinkTypeNowhere
		return
	}

	// Parse display name
	form.DisplayName = r.PostForm.Get("displayname")
	if len(form.DisplayName) > maxNameLen {
		nameTooLong := fmt.Sprintf("Display name must be no more than %d characters", maxNameLen)
		errors = append(errors, nameTooLong)
	}

	// Parse social media link
	form.SocialLinkType = r.PostForm.Get("linktype")
	switch form.SocialLinkType {
	case common.SocialLinkTypeNowhere:
		form.SocialLink = ""
	case common.SocialLinkTypeTwitter:
		twitterUsername := r.PostForm.Get("twittername")
		usernameLen := len(twitterUsername)
		if usernameLen < minUsernameLen || usernameLen > maxUsernameLen {
			lenError := fmt.Sprintf("Twitter username must be between %d and %d characters",
				minUsernameLen, maxUsernameLen)
			errors = append(errors, lenError)
			break
		}
		twitterUsername = strings.Replace(twitterUsername, "@", "", 1)
		form.SocialLink = url.QueryEscape(twitterUsername)
	case common.SocialLinkTypeInstagram:
		instagramUsername := r.PostForm.Get("instagramname")
		usernameLen := len(instagramUsername)
		if usernameLen < minUsernameLen || usernameLen > maxUsernameLen {
			lenError := fmt.Sprintf("Instagram username must be between %d and %d characters",
				minUsernameLen, maxUsernameLen)
			errors = append(errors, lenError)
			break
		}
		instagramUsername = strings.Replace(instagramUsername, "@", "", 1)
		form.SocialLink = url.QueryEscape(instagramUsername)
	case common.SocialLinkTypeCustom:
		customLink := r.PostForm.Get("customlink")
		customLinkLen := len(customLink)
		if customLinkLen > maxCustomLinkLen {
			lenError := fmt.Sprintf("Custom link must be no more than %d characters", maxCustomLinkLen)
			errors = append(errors, lenError)
			break
		}
		if !strings.HasPrefix(customLink, "http://") &&
			!strings.HasPrefix(customLink, "https://") {
			errors = append(errors, "Custom link must begin with https:// or http://")
			break
		}
		normalized, err := normalizeLink(customLink)
		if err != nil {
			errors = append(errors, "Error parsing custom link. Please check it and try again.")
			break
		}
		form.SocialLink = normalized
	default:
		errors = append(errors, "Error parsing link")
	}

	// Parse message
	form.Message = r.PostForm.Get("msg")
	messageLen := len(form.Message)
	if messageLen > maxMessageLen {
		lenError := fmt.Sprintf("Message must be no more than %d characters", messageLen)
		errors = append(errors, lenError)
	}

	// Parse canvas data
	form.ImgKind = imgKind
	iconError := "Error processing icon data"
	encodedCanvasData := r.PostForm.Get("canvasdata")
	form.ImgData, err = base64.StdEncoding.DecodeString(encodedCanvasData)
	imgDataLen := len(form.ImgData)
	if err != nil {
		errors = append(errors, iconError)
	} else if imgDataLen != 0 && imgDataLen != expectedIconDataLen {
		errors = append(errors, iconError)
	}
	return
}
