package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/peteraba/cli-to-http/convert"
)

type options struct {
	url         string
	method      string
	header      http.Header
	input       string
	output      string
	returnCode  bool
	verbose     bool
	cipherKey   string
	cipherType  string
	encryptType string
	paddingType string
	encode      string
	decode      string
}

func main() {
	o := parseFlags()

	body, err := read(o)
	check(err, o, 0)
	o, body, err = parseInput(o, body)

	printInput(o.verbose, o.input, o.method, o.url, body, o.header)

	if o.url == "" {
		if o.verbose {
			printFirstLineChars(body)
		}
		check(errors.New("URL must be provided"), o, 0)
	}

	var encrypter *convert.BlockEncrypter
	if o.cipherType != "" {
		encrypter = convert.NewBlockEncrypter([]byte(o.cipherKey), o.cipherType, o.encryptType, o.paddingType)
	}

	if o.encode != "" || encrypter != nil {
		body, err = encodeBody(body, o.encode, encrypter)
		check(err, o, 0)
		printInput(o.verbose, o.input, o.method, o.url, body, o.header)
	}

	code, resp, err := makeRequest(o.method, o.url, body, o.header)
	check(err, o, code)

	if o.decode != "" || encrypter != nil {
		resp, err = decodeBody(resp, o.decode, encrypter)
		check(err, o, 0)
		printOutput(o.verbose, o.output, code, resp)
	}

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
		headers    string
		encryption string
		o          options
	)

	o.header = http.Header{}

	flag.StringVar(&o.url, "url", "", "url to send the request to")
	flag.StringVar(&o.method, "method", "POST", "http method to use")
	flag.StringVar(&headers, "headers", "", "comma separated list of headers in key:value format. example: Content-Type:application/xml,Accept:application/xml")
	flag.StringVar(&o.input, "input", "request.txt", "file to read and parse for request data. read stdin if empty is provided")
	flag.StringVar(&o.output, "output", "response.txt", "file to fill with response data. write to stdout if empty is provided")
	flag.BoolVar(&o.returnCode, "exit_as_return_code", false, "exit code will be the same as return code if true. if false the return code will be the first line of the output")
	flag.BoolVar(&o.verbose, "verbose", false, "output debugging information on standard output")
	flag.StringVar(&o.cipherKey, "cipher_key", "", "cipher key to use for encryption and decryption")
	flag.StringVar(&encryption, "encrypt", "", "encryption algorithms to use as listed on https://8gwifi.org/CipherFunctions.jsp. Only AES/ECB/PKCS5PADDING is supported at the moment.")
	flag.StringVar(&o.encode, "encode", "", "encoding algorithm to apply. will be applied after encryption. Only base64 is supported at the moment")
	flag.StringVar(&o.decode, "decode", "", "decoding algorithm to apply. will be applied before decryption. Only base64 is supported at the moment")

	flag.Parse()

	o.header = addHeaders(o.header, headers)
	o.cipherType, o.encryptType, o.paddingType = splitEncryption(encryption)

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
		line     string
	)

	lines := strings.Split(string(body), "\n")

	r := regExp()
	for bodyFrom, line = range lines {
		matches := r.FindStringSubmatch(line)
		if len(matches) == 0 {
			break
		}

		doubleBreak := false
		switch matches[1] {
		case "URL":
			o.url = matches[2]
			break
		case "METHOD":
			o.method = matches[2]
			break
		case "HEADERS":
			o.header = addHeaders(o.header, matches[2])
			break
		case "EXIT_AS_RETURN_CODE":
			o.returnCode = false
			if matches[2] == "1" || strings.ToLower(matches[2]) == "true" {
				o.returnCode = true
			}
			break
		case "VERBOSE":
			o.verbose = false
			if matches[2] == "1" || strings.ToLower(matches[2]) == "true" {
				o.verbose = true
			}
			break
		case "CIPHER_KEY":
			o.cipherKey = matches[2]
			break
		case "ENCRYPTION":
			o.cipherType, o.encryptType, o.paddingType = splitEncryption(matches[2])
			break
		default:
			doubleBreak = true
		}

		if doubleBreak {
			break
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

func encodeBody(body []byte, encode string, encrypter *convert.BlockEncrypter) ([]byte, error) {
	if encrypter != nil {
		body = encrypter.Encrypt(body)
	}

	if encode == "base64" {
		body = []byte(base64.StdEncoding.EncodeToString(body))
	}

	return body, nil
}

func decodeBody(body []byte, decode string, encrypter *convert.BlockEncrypter) ([]byte, error) {
	var err error

	if encrypter != nil {
		body = encrypter.Decrypt(body)
	}

	if decode == "base64" {
		body, err = base64.StdEncoding.DecodeString(string(body))
		if err != nil {
			return []byte{}, err
		}
	}

	return body, nil
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

func addHeaders(header http.Header, headers string) http.Header {
	for _, h := range strings.Split(headers, ",") {
		if h == "" {
			break
		}

		parts := strings.Split(h, ":")
		header.Add(parts[0], parts[1])
	}

	return header
}

func splitEncryption(encryption string) (string, string, string) {
	parts := strings.Split(encryption, "/")
	if len(parts) > 2 {
		p2, l2 := parts[2], len(parts[2])
		if l2 > 7 && p2[l2-7:] == "PADDING" {
			return parts[0], parts[1], p2[:l2-7]
		}
		return parts[0], parts[1], p2
	}
	if len(parts) > 1 {
		return parts[0], parts[1], ""
	}
	if len(parts) > 0 {
		return parts[0], "", ""
	}

	return "", "", ""
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

func printFirstLineChars(body []byte) {
	lines := strings.Split(string(body), "\n")

	r := regExp()
	for _, line := range lines {
		matches := r.FindStringSubmatch(line)
		if len(matches) == 0 {
			break
		}

		fmt.Println(line)
		for i, elem := range line {
			fmt.Println(i, elem, string(elem))
		}
	}
}

func regExp() *regexp.Regexp {
	return regexp.MustCompile("([A-Z_]+)=([a-zA-Z0-9.,: /-]+)\\s*$")
}
