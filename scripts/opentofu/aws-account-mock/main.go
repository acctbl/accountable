package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"
)

func main() {
	accountFile := flag.String("account-file", "", "file containing the AWS account ID returned by the mock")
	endpointFile := flag.String("endpoint-file", "", "file receiving the local endpoint URL")
	flag.Parse()
	if *accountFile == "" || *endpointFile == "" {
		fmt.Fprintln(os.Stderr, "usage: aws-account-mock --account-file <path> --endpoint-file <path>")
		os.Exit(2)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	endpoint := "http://" + listener.Addr().String()
	if err := os.WriteFile(*endpointFile, []byte(endpoint), 0o600); err != nil {
		panic(err)
	}

	handler := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		accountPayload, err := os.ReadFile(*accountFile)
		if err != nil || len(accountPayload) != 12 {
			http.Error(response, "invalid offline account file", http.StatusInternalServerError)
			return
		}
		accountID := string(accountPayload)
		if err := request.ParseForm(); err != nil {
			http.Error(response, err.Error(), http.StatusBadRequest)
			return
		}
		response.Header().Set("Content-Type", "text/xml")
		switch request.Form.Get("Action") {
		case "GetUser":
			if _, err := fmt.Fprintf(response, `<?xml version="1.0" encoding="UTF-8"?>
<GetUserResponse xmlns="https://iam.amazonaws.com/doc/2010-05-08/"><GetUserResult><User><Path>/</Path><UserName>offline</UserName><UserId>AIDAOFFLINE</UserId><Arn>arn:aws:iam::%s:user/offline</Arn><CreateDate>2020-01-01T00:00:00Z</CreateDate></User></GetUserResult><ResponseMetadata><RequestId>offline</RequestId></ResponseMetadata></GetUserResponse>`, accountID); err != nil {
				return
			}
		case "GetCallerIdentity":
			if _, err := fmt.Fprintf(response, `<?xml version="1.0" encoding="UTF-8"?>
<GetCallerIdentityResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/"><GetCallerIdentityResult><Arn>arn:aws:iam::%s:user/offline</Arn><UserId>AIDAOFFLINE</UserId><Account>%s</Account></GetCallerIdentityResult><ResponseMetadata><RequestId>offline</RequestId></ResponseMetadata></GetCallerIdentityResponse>`, accountID, accountID); err != nil {
				return
			}
		default:
			response.WriteHeader(http.StatusBadRequest)
			if _, err := io.WriteString(response, `<?xml version="1.0" encoding="UTF-8"?><ErrorResponse><Error><Type>Sender</Type><Code>InvalidAction</Code><Message>unsupported offline action</Message></Error><RequestId>offline</RequestId></ErrorResponse>`); err != nil {
				return
			}
		}
	})
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       2 * time.Second,
		WriteTimeout:      2 * time.Second,
		IdleTimeout:       2 * time.Second,
	}
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}
