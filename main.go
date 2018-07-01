package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
)

type options struct {
	url        string
	method     string
	header     http.Header
	input      string
	output     string
	returnCode bool
	verbose    bool
}

func main() {
	o := parseFlags()

	body, err := read(o)
	check(err, o, 0)
	o, body, err = parseInput(o, body)

	printInput(o.verbose, o.input, o.method, o.url, body, o.header)

	if o.url == "" {
		check(errors.New("URL must be provided"), o, 0)
	}

	code, resp, err := makeRequest(o.method, o.url, body, o.header)
	check(err, o, code)

	err = write(o, code, resp)
	check(err, o, code)

	printOutput(o.verbose, o.output, code, resp)

	if o.returnCode {
		os.Exit(code)
	}
}

func check(e error, o options, code int) {
	if e == nil {
		return
	}
	if o.returnCode {
		fmt.Println(e)
		os.Exit(code)
	}
	panic(e)
}

func parseFlags() options {
	var (
		headers string
		o       options
	)

	o.header = http.Header{}

	flag.StringVar(&o.url, "url", "", "url to send the request to")
	flag.StringVar(&o.method, "method", "POST", "http method to use")
	flag.StringVar(&headers, "headers", "", "comma separated list of headers in key:value format. example: Content-Type:application/xml,Accept:application/xml")
	flag.StringVar(&o.input, "input", "request.txt", "file to read and parse for request data. read stdin if empty is provided")
	flag.StringVar(&o.output, "output", "response.txt", "file to fill with response data. write to stdout if empty is provided")
	flag.BoolVar(&o.returnCode, "exit_as_return_code", false, "exit code will be the same as return code if true. if false the return code will be the first line of the output")
	flag.BoolVar(&o.verbose, "verbose", false, "output debugging information on standard output")

	flag.Parse()

	o = addHeaders(o, headers)

	return o
}

func read(o options) ([]byte, error) {
	var text string

	if o.input == "" {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			text += scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			return []byte{}, err
		}
	} else {
		if _, err := os.Stat(o.input); os.IsNotExist(err) {
			return []byte{}, err
		}
		data, err := ioutil.ReadFile(o.input)
		if err != nil {
			return []byte{}, err
		}
		text = string(data)
	}

	return []byte(text), nil
}

func parseInput(o options, body []byte) (options, []byte, error) {
	var (
		bodyFrom int
		line string
	)

	lines := strings.Split(string(body), "\n")

	r := regexp.MustCompile("^([A-Z]+)=([a-zA-Z0-9.,: /-]+)$")
	for bodyFrom, line = range lines {
		matches := r.FindStringSubmatch(line)
		fmt.Printf("line: %s, matches: %v\n", line, matches)
		if len(matches) == 0 {
			break
		}

		doubleBreak := false
		switch matches[1] {
		case "URL":
			o.url = matches[2]
			break;
		case "METHOD":
			o.method = matches[2]
			break;
		case "HEADERS":
			o = addHeaders(o, matches[2])
			break;
		case "EXIT_AS_RETURN_CODE":
			o.returnCode = false
			if matches[2] == "1" || strings.ToLower(matches[2]) == "true" {
				o.returnCode = true
			}
			break;
		case "VERBOSE":
			o.verbose = false
			if matches[2] == "1" || strings.ToLower(matches[2]) == "true" {
				o.verbose = true
			}
			break;
		default:
			doubleBreak = true
		}

		if doubleBreak {
			break;
		}
	}

	str := strings.Join(lines[bodyFrom:], "\n")

	return o, []byte(str), nil
}

func write(o options, code int, resp []byte) error {
	if !o.returnCode {
		resp = []byte(fmt.Sprintf("%d\n%s", code, string(resp)))
	}
	if o.output == "" {
		fmt.Println(string(resp))
	}
	return ioutil.WriteFile(o.output, resp, 0644)
}

func makeRequest(method, url string, b []byte, header http.Header) (int, []byte, error) {
	req, err := http.NewRequest(method, url, bytes.NewReader(b))
	if err != nil {
		return 0, []byte{}, err
	}

	req.Header = header

	client := &http.Client{}
	resp, err := client.Do(req)

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	return resp.StatusCode, body, err
}

func addHeaders(o options, headers string) options {
	for _, h := range strings.Split(headers, ",") {
		if h == "" {
			break
		}

		parts := strings.Split(h, ":")
		o.header.Add(parts[0], parts[1])
	}

	return o
}

func printInput(verbose bool, input, method, url string, req []byte, header http.Header) {
	if !verbose {
		return
	}

	fmt.Printf("INPUT Data\n")
	fmt.Printf("----------\n")
	fmt.Printf("Request file: %s\n", input)
	fmt.Printf("HTTP Method: %s\n", method)
	fmt.Printf("URL: %s\n", url)
	fmt.Printf("Header: %s\n", header)
	fmt.Printf("Request Body:\n")
	if len(req) > 0 {
		fmt.Printf("%s\n", string(req))
	}
	fmt.Println()
}

func printOutput(verbose bool, output string, code int, resp []byte) {
	if !verbose {
		return
	}

	fmt.Printf("OUTPUT Data\n")
	fmt.Printf("-----------\n")
	fmt.Printf("Response file: %s\n", output)
	fmt.Printf("Response Code: %d\n", code)
	fmt.Printf("Response Body:\n")
	if len(resp) > 0 {
		fmt.Printf("%s\n", string(resp))
	}
	fmt.Println()
}
