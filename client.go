package gosiklu

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/go-resty/resty/v2"
	"log"
	"net/url"
	"strings"
	"time"
)

const DefaultSikluTimeout = 10 * time.Second

type CommandReply struct {
	Request string   `xml:"request"`
	EndCode []string `xml:"end-code"`
	Text    []string `xml:"text"`
}

type Client struct {
	Host                  string
	User                  string
	password, encodedPass string
	logger                clientLogger
	httpClient            *resty.Client
	debugMode             bool
	usesPasswordEncode    bool
	cmdsAsBase64          bool
	ctx                   context.Context
}

var (
	ErrLoginFailed = errors.New("failed to login: no authentication token returned")
)

func (c CommandReply) AllWorked() bool {
	for _, e := range c.EndCode {
		if e != "ok" {
			return false
		}
	}
	return true
}

func (c *Client) SetDebug(debug bool) *Client {
	c.httpClient.SetDebug(debug)
	c.debugMode = debug
	return c
}

// New creates a new Client and logs into the radio
// If login or access to the radio are unsuccessful, nil is returned
// If successful, a pointer to the Client is returned and the caller is responsible for calling Close() when done
// The host parameter should be an IP address or hostname of the radio
func New(ctx context.Context, host, user, pass string) (*Client, error) {
	c := &Client{
		Host:        host,
		User:        user,
		password:    pass,
		encodedPass: passwordEncode(pass),
		logger:      clientLogger{},
		ctx:         ctx,
	}
	// Siklu web server only does an insecure cipher, so we have to force that here
	// also need to skip verification of certificates and allow older TLS versions
	c.httpClient = resty.New().SetTLSClientConfig(&tls.Config{
		CipherSuites: []uint16{
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_AES_256_GCM_SHA384,
		},
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS11,
		MaxVersion:         tls.VersionTLS13,
	}).SetLogger(&c.logger)
	if err := c.login(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client) login() error {
	// Warning: you cannot use the normal SetQueryParams() methods because they take a map[string]string and the Siklu
	// web server is sensitive to the order of the query parameters.  You cannot use SetQueryString() because it
	// just parses the string with url.ParseQuery() which does not preserve the order of the query parameters.
	// You have to supply the `caller_url=/` parameter else the Siklu web server will never return on the HTTP call
	newContext, cancel := context.WithTimeout(c.ctx, DefaultSikluTimeout)
	defer cancel()
	resp, err := c.httpClient.R().
		SetContext(newContext).Get(fmt.Sprintf("https://%s/", c.Host))
	if err != nil {
		return errors.Join(errors.New("unable to reach radio"), err)
	}
	if resp.StatusCode() != 200 {
		return fmt.Errorf("unknown response: %v", resp.Status())
	}
	pass := url.PathEscape(c.password)
	// See if the resulting page contains the string "PasswordEncode"
	if bytes.Contains(resp.Body(), []byte("function PasswordEncode")) {
		// Maybe an EH1200/710 which takes the password as encoded but commands as plaintext
		c.usesPasswordEncode = true
		c.cmdsAsBase64 = false
		pass = c.encodedPass
	} else if bytes.Contains(resp.Body(), []byte("EH-614TX")) {
		// EH614 which takes the password and commands as plaintext
		c.usesPasswordEncode = false
		c.cmdsAsBase64 = false
	} else {
		// Maybe an EH8100 which takes the password as plaintext but commands as base64
		c.usesPasswordEncode = false
		c.cmdsAsBase64 = true
	}
	resp, err = c.httpClient.R().
		SetContext(newContext).
		SetBody(fmt.Sprintf(`user=%s&password=%s&caller_url=%%2F`, url.PathEscape(c.User), pass)).
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetHeader("accept", "*/*").
		Post(fmt.Sprintf("https://%s/main/handleform", c.Host))
	if err != nil {
		return errors.Join(errors.New("unable to reach radio"), err)
	}
	if resp.StatusCode() != 200 {
		return fmt.Errorf("failed to login: %v", resp.Status())
	}
	if cookies := resp.Cookies(); cookies != nil {
		for _, cookie := range cookies {
			if cookie.Name == "auth_cookie" || cookie.Name == "app_auth_cookie" {
				return nil
			}
		}
	}
	return errors.New("failed to login: no authentication token returned")
}

func (c *Client) Close() {
	newContext, cancel := context.WithTimeout(c.ctx, DefaultSikluTimeout)
	defer cancel()
	resp, err := c.httpClient.R().
		SetContext(newContext).
		SetHeader("accept", "*/*").Get(fmt.Sprintf("https://%s/main/logout", c.Host))
	if err == nil && c.debugMode {
		log.Printf(resp.Status())
	}
}

func (c *Client) SaveRunning() error {
	var (
		reply CommandReply
		err   error
	)
	reply, err = c.Command([]string{"copy running-configuration startup-configuration"})
	if err != nil {
		return err
	}
	if !reply.AllWorked() {
		return errors.New(reply.Text[0])
	}
	return nil
}

func (c *Client) GetInfo(sections []string) (SikluData, error) {
	var data SikluData

	if len(sections) == 0 {
		return data, errors.New("no sections specified")
	}

	// You cannot use SetQueryParams() because it will not preserve the order of the query parameters
	// so we manually build the query string
	q := fmt.Sprintf("https://%s/main/web.cgi?%s", c.Host,
		url.PathEscape("mo-info "+strings.Join(sections, " ; ")))
	newContext, cancel := context.WithTimeout(c.ctx, DefaultSikluTimeout)
	defer cancel()
	resp, err := c.httpClient.R().
		SetContext(newContext).
		SetHeader("accept", "*/*").Get(q)
	if err != nil {
		return data, err
	}
	if resp.StatusCode() != 200 {
		return data, fmt.Errorf("%d %s", resp.StatusCode(), resp.Status())
	}
	return c.parseReply(resp.Body())
}

// parseReply parses the returned XML from the radio into a SikluData struct
// any common fix ups can be done here
func (c *Client) parseReply(data []byte) (SikluData, error) {
	var d SikluData
	err := xml.Unmarshal(data, &d)
	if err != nil {
		return d, fmt.Errorf("XML parse error: %v", err)
	}
	return d, nil
}

// Command sends one or more CLI commands to the radio and returns the results.
// The reply.Request will contain the concatenated list of all the commands
// The reply.EndCode slice will contain a result for each commmand ("ok" or "error")
// The reply.Text slice will contain the CLI result for each command
// Examples:
//
//	Request: "simple-command set event-cfg cinr-out-of-range alarm-mask no ; set event-cfg temperature-high alarm-mask no",
//	   EndCode: ([]string) {"ok", "ok" }
//	   Text: ([]string) {
//	      "\nSet done: event-cfg cinr-out-of-range",
//	      "\nSet done: event-cfg temperature-high"
//	   }
//
//	Request: "simple-command set event-cfg cinr-out-of-range alarm-mask no ; set event-cfg temperatureHigh alarm-mask no",
//	   EndCode: ([]string) {"ok", "error" }
//	   Text: ([]string) {
//	      "\nSet done: event-cfg cinr-out-of-range",
//	      "\n% Ambiguous command:  set event-cfg ?\n\nset event-cfg <event-cfg-id-list>  [trap-mask <value>] [alarm-mask <value>] [threshold-high <value>] [threshold-low <value>] [hysteresis <value>] [mask <value>]\n    <event-cfg-id-list>    : list:  | temperature-high | cfm-fault-alarm | loopback-enabled | tx-mute-enabled | ql-eec1-or-worse | cold-start | modulation-change | sfp-in | ref-clock-switch | cinr-out-of-range | rssi-out-of-range | lowest-modulation | pse-voltage | rate-change\n"
//	   }
func (c *Client) Command(cmds []string) (CommandReply, error) {
	var (
		data CommandReply
		q    string
	)

	if c.cmdsAsBase64 {
		// Concat all together and then base64 encode
		var all string
		for i, cmd := range cmds {
			if i > 0 {
				all += ";"
			}
			all += cmd
		}
		q = fmt.Sprintf("https://%s/main/web.cgi?simple-command%%20%s", c.Host, base64.StdEncoding.EncodeToString([]byte(all)))
	} else {
		q = fmt.Sprintf("https://%s/main/web.cgi?simple-command%%20", c.Host)
		for i, cmd := range cmds {
			if i > 0 {
				q += ";"
			}
			q += url.PathEscape(cmd)
		}
	}
	newContext, cancel := context.WithTimeout(c.ctx, DefaultSikluTimeout)
	defer cancel()
	resp, err := c.httpClient.R().
		SetContext(newContext).
		SetHeader("accept", "*/*").Get(q)
	if err != nil {
		return data, err
	}
	if resp.StatusCode() != 200 {
		return data, fmt.Errorf("%d %s", resp.StatusCode(), resp.Status())
	}
	b := resp.Body()
	if err = xml.Unmarshal(b, &data); err != nil {
		return data, fmt.Errorf("XML parse error: %v", err)
	}
	return data, nil
}
