package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func authenticate(client *http.Client) AuthenticatedClient {
	initResponse, _ := client.Get("https://room.chuo-u.ac.jp/ct/")
	samlInitResponse, _ := client.Get(initResponse.Header.Get("Location"))
	loginQuery, _ := url.ParseQuery(strings.Split(samlInitResponse.Header.Get("Location"), "?")[1])
	backURL := "https://gakunin-idp.c.chuo-u.ac.jp" + loginQuery.Get("back")
	samlSessionId := parseSetCookieHeaders(samlInitResponse.Header.Values("Set-Cookie"))["SimpleSAMLSessionID"]
	loginPageResponse, _ := client.Get("https://gakunin-idp.c.chuo-u.ac.jp/" + samlInitResponse.Header.Get("Location"))
	loginPage, _ := goquery.NewDocumentFromReader(loginPageResponse.Body)
	loginSessionId, _ := loginPage.Find("#sessid").Attr("value")

	loginForm := url.Values{}
	loginForm.Set("username", USERNAME)
	loginForm.Set("password", PASSWORD)
	loginForm.Set("sessid", loginSessionId)
	loginForm.Set("op", "login")
	loginResponse, _ := client.PostForm("https://gakunin-idp.c.chuo-u.ac.jp/pub/login.cgi", loginForm)
	authTicket := parseSetCookieHeaders(loginResponse.Header.Values("Set-Cookie"))["auth_tkt"]

	samlRequest, _ := http.NewRequest("GET", backURL, nil)
	samlRequest.Header.Set("Cookie", fmt.Sprintf("SimpleSAMLSessionID=%s; auth_tkt=%s", samlSessionId, authTicket))
	samlResponse, _ := client.Do(samlRequest)
	samlPage, _ := goquery.NewDocumentFromReader(samlResponse.Body)
	samlResponseData, _ := samlPage.Find("input[name=SAMLResponse]").Attr("value")
	relayState, _ := samlPage.Find("input[name=RelayState").Attr("value")
	samlForm := url.Values{}
	samlForm.Set("SAMLResponse", samlResponseData)
	samlForm.Set("RelayState", relayState)
	shibbolethResponse, _ := client.PostForm("https://room.chuo-u.ac.jp/Shibboleth.sso/SAML2/POST", samlForm)
	shibbolethCookies := parseSetCookieHeaders(shibbolethResponse.Header.Values("Set-Cookie"))
	var sessionCookieName, sessionId string
	for cookieName, cookieValue := range shibbolethCookies {
		if strings.HasPrefix(cookieName, "_shibsession_") {
			sessionCookieName = cookieName
			sessionId = cookieValue
		}
	}
	return AuthenticatedClient{client, sessionCookieName, sessionId}
}

type AuthenticatedClient struct {
	Client            *http.Client
	SessionCookieName string
	SessionId         string
}

func (ac AuthenticatedClient) Do(req *http.Request) (*http.Response, error) {
	sessionCookie := http.Cookie{}
	sessionCookie.Name = ac.SessionCookieName
	sessionCookie.Value = ac.SessionId
	req.AddCookie(&sessionCookie)
	return ac.Client.Do(req)
}

func (ac AuthenticatedClient) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return ac.Do(req)
}
