package main

import (
	"html/template"
)

type HomePageData struct {
	WSOrigin     string
	InitialNines string
	InitialGoal  string
}

type LeavingPageData struct {
	URL string
}

type ErrorPageData struct {
	ErrorCode int
}

type CardPageData struct {
	PublicToken string
	PayResult   string
	Paid        bool
	MailToLink  string
	TwitterLink string
	ShareLink   string
	CSRFField   template.HTML
}

type ThankYouPageData struct {
	MailToLink  string
	TwitterLink string
	ShareLink   string
}
