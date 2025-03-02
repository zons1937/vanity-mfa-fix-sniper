package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/valyala/fasthttp"
)

var (
	token    = ""
	pass     = ""
	mfaToken string
)

type VanityResponse struct {
	MFA struct {
		Ticket string `json:"ticket"`
	} `json:"mfa"`
}

type MFAResponse struct {
	Token string `json:"token"`
}

type RestartRequest struct {
	Restart bool `json:"restart"`
}

func setCommonHeaders(req *fasthttp.Request, token string) {
	req.Header.Set("Authorization", token)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "+
		"AppleWebKit/537.36 (KHTML, like Gecko) discord/1.0.9164 "+
		"Chrome/124.0.6367.243 Electron/30.2.0 Safari/537.36")
	req.Header.Set("X-Super-Properties", "eyJvcyI6IldpbmRvd3MiLCJicm93c2VyIjoiRGlzY29yZCBDbGllbnQiLCJyZWxlYXNlX2NoYW5uZWwiOiJwdGIiLCJjbGllbnRfdmVyc2lvbiI6IjEuMC4xMTMwIiwib3NfdmVyc2lvbiI6IjEwLjAuMTkwNDUiLCJvc19hcmNoIjoieDY0IiwiYXBwX2FyY2giOiJ4NjQiLCJzeXN0ZW1fbG9jYWxlIjoidHIiLCJoYXNfY2xpZW50X21vZHMiOmZhbHNlLCJicm93c2VyX3VzZXJfYWdlbnQiOiJNb3ppbGxhLzUuMCAoV2luZG93cyBOVCAxMC4wOyBXaW42NDsgeDY0KSBBcHBsZVdlYktpdC81MzcuMzYgKEtIVE1MLCBsaWtlIEdlY2tvKSBkaXNjb3JkLzEuMC4xMTMwIENocm9tZS8xMjguMC42NjEzLjE4NiBFbGVjdHJvbi8zMi4yLjcgU2FmYXJpLzUzNy4zNiIsImJyb3dzZXJfdmVyc2lvbiI6IjMyLjIuNyIsIm9zX3Nka192ZXJzaW9uIjoiMTkwNDUiLCJjbGllbnRfYnVpbGRfbnVtYmVyIjozNjY5NTUsIm5hdGl2ZV9idWlsZF9udW1iZXIiOjU4NDYzLCJjbGllbnRfZXZlbnRfc291cmNlIjpudWxsfQ==")
	req.Header.Set("X-Discord-Timezone", "Europe/Istanbul")
	req.Header.Set("X-Discord-Locale", "en-US")
	req.Header.Set("X-Debug-Options", "bugReporterEnabled")
	req.Header.Set("Content-Type", "application/json")
}

func handleMFA(token, pass string) string {
	client := &fasthttp.Client{}
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	
	req.SetRequestURI("https://discord.com/api/v9/guilds/0/vanity-url")
	req.Header.SetMethod("PATCH")
	setCommonHeaders(req, token)
	
	err := client.Do(req, resp)
	if err != nil || resp.StatusCode() != fasthttp.StatusUnauthorized {
		return "err"
	}
	
	var vanityResponse VanityResponse
	if err := json.Unmarshal(resp.Body(), &vanityResponse); err != nil {
		return "err"
	}
	
	return sendMFA(token, vanityResponse.MFA.Ticket, pass)
}

func sendMFA(token, ticket, pass string) string {
	log.Printf("ticket: %s, sifre : pass: %s ile mfa token aliyorum", ticket, pass)
	
	payload := struct {
		Ticket string `json:"ticket"`
		Type   string `json:"mfa_type"`
		Data   string `json:"data"`
	}{
		Ticket: ticket,
		Type:   "password",
		Data:   pass,
	}
	
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling payload to JSON: %v", err)
		return "err"
	}
	
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	
	req.SetRequestURI("https://discord.com/api/v9/mfa/finish")
	req.Header.SetMethod("POST")
	setCommonHeaders(req, token)
	req.SetBody(jsonPayload)
	
	client := &fasthttp.Client{}
	err = client.Do(req, resp)
	if err != nil {
		log.Printf("Network error: %v", err)
		return "err"
	}
	
	if resp.StatusCode() != fasthttp.StatusOK {
		log.Printf("Error response from server: %d", resp.StatusCode())
		return "err"
	}
	
	var mfaResponse MFAResponse
	if err := json.Unmarshal(resp.Body(), &mfaResponse); err != nil {
		log.Printf("Error unmarshalling response: %v", err)
		return "err"
	}
	
	log.Printf("mfa tokeni aldim: %s", mfaResponse.Token)
	return mfaResponse.Token
}

func sendMfaTokenServer(mfaToken string) {
	data, _ := json.Marshal(map[string]string{
		"mfaToken": mfaToken,
	})
	
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	
	req.SetRequestURI("http://localhost:6931/duckevilsontop")
	req.Header.SetMethod("POST")
	req.SetBody(data)
	req.Header.Set("Content-Type", "application/json")
	
	client := &fasthttp.Client{}
	err := client.Do(req, resp)
	if err != nil {
		fmt.Println("Error sending request to server:", err)
		return
	}
	
	fmt.Println("sunucu cevabi:", resp.StatusCode())
}

func restartHandler(ctx *fasthttp.RequestCtx) {
	if ctx.IsPost() && string(ctx.Path()) == "/restart" {
		var reqData RestartRequest
		if err := json.Unmarshal(ctx.PostBody(), &reqData); err != nil {
			ctx.Error("Gecersiz JSON", fasthttp.StatusBadRequest)
			return
		}
		
		if reqData.Restart {
			go sendMfaTokenServer(mfaToken)
			ctx.SetStatusCode(fasthttp.StatusOK)
			ctx.SetBody([]byte(`{"message": "mfa token gonderiyorum."}`))
			return
		}
	}
	
	ctx.Error("Bulunamadi", fasthttp.StatusNotFound)
}

func main() {
	fmt.Println("sunucuyu baslatiyorumm")
	go func() {
		if err := fasthttp.ListenAndServe(":8000", restartHandler); err != nil {
			fmt.Println("Restart server error:", err)
		}
	}()
	
	fmt.Println("mfa token aliyorum")
	mfaToken = handleMFA(token, pass)
	if mfaToken != "err" {
		fmt.Println("mfa tokeni gonderiyorum")
		sendMfaTokenServer(mfaToken)
	} else {
		fmt.Println("mfa basarisiz oldu gonderemiyorum")
	}
	
	ticker := time.NewTicker(50 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		fmt.Println("mfa token suresi gecti yenisini aliyom")
		mfaToken = handleMFA(token, pass)
		if mfaToken != "err" {
			fmt.Println("yeni mfa tokeni gonderiyorum")
			sendMfaTokenServer(mfaToken)
		} else {
			fmt.Println("mfa gondermekte hata olustu gonderemedim")
		}
	}
}