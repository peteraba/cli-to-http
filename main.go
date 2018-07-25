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
	"reflect"
	"regexp"
	"strings"

	"github.com/peteraba/cli-to-http/convert"
	"hash/crc32"
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
	mode        string
}

func main() {
	o, body := collectOptions()

	encrypter := getEncrypter(o)

	body = encryptEncode(o, encrypter, body)

	code, resp := handleRequest(o, body)

	resp = decodeDecrypt(o, encrypter, code, resp)

	handleOutput(o, code, resp)
}

func collectOptions() (options, []byte) {
	o := parseFlags()

	body, err := read(o)
	check(err, o, 0)

	o, body, err = parseInput(o, body)
	check(err, o, 0)

	printOptions(o)
	printInput(o.verbose, o.input, o.method, o.url, body, o.header, []string{}, false)

	return o, body
}

func getEncrypter(o options) *convert.BlockEncrypter {
	if o.cipherType != "" {
		return convert.NewBlockEncrypter([]byte(o.cipherKey), o.cipherType, o.encryptType, o.paddingType)
	}

	return nil
}

func encryptEncode(o options, encrypter *convert.BlockEncrypter, body []byte) []byte {
	var err error

	extra := []string{fmt.Sprintf(`Original: "%s"\n`, string(body))}
	body = []byte(strings.TrimSpace(string(body)))

	if o.encode == "" && encrypter == nil {
		return body
	}

	extra = append(extra, fmt.Sprintf(`Trimmed: "%s"\n`, string(body)))

	if encrypter != nil {
		body = encrypter.Encrypt(body)
	}

	encodings := strings.Split(o.encode, ",")
	for _, encoding := range encodings {
		switch encoding {
		case "base64":
			body = []byte(base64.StdEncoding.EncodeToString(body))
			extra = append(extra, fmt.Sprintf(`Base64 encoded: "%s"\n`, string(body)))
			break
		case "crc32":
			body = []byte(fmt.Sprint(crc32.ChecksumIEEE([]byte(body))))
			extra = append(extra, fmt.Sprintf(`Crc32 checksum: "%s"\n`, string(body)))
			break
		}
	}

	check(err, o, 0)
	printInput(o.verbose, o.input, o.method, o.url, body, o.header, extra, true)

	return body
}

func handleRequest(o options, body []byte) (int, []byte) {
	if o.mode == "encode" || o.mode == "decode" {
		return 0, body
	}

	if o.mode != "send" {
		panic("only send, encode and decode modes are supported")
	}

	if o.url == "" {
		if o.verbose {
			printFirstLineChars(body)
		}
		check(errors.New("URL must be provided"), o, 0)
	}

	code, resp, err := makeRequest(o.method, o.url, body, o.header)
	check(err, o, code)

	return code, resp
}

func decodeDecrypt(o options, encrypter *convert.BlockEncrypter, code int, resp []byte) []byte {
	var err error

	if o.decode == "" && encrypter == nil {
		return resp
	}

	resp = []byte(strings.TrimSpace(string(resp)))

	printOutput(o.verbose, o.output, code, resp, true)

	if o.decode == "base64" {
		resp, err = base64.StdEncoding.DecodeString(string(resp))
	}

	if encrypter != nil {
		resp = encrypter.Decrypt(resp)
	}

	check(err, o, 0)

	return resp
}

func handleOutput(o options, code int, resp []byte) {
	err := write(o, code, resp)
	check(err, o, code)

	printOutput(o.verbose, o.output, code, resp, false)

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
	flag.StringVar(&o.mode, "mode", "send", "execution mode. (send, encode, decode)")
	flag.StringVar(&o.method, "method", "POST", "http method to use")
	flag.StringVar(&headers, "headers", "", "comma separated list of headers in key:value format. example: Content-Type:application/xml,Accept:application/xml")
	flag.StringVar(&o.input, "input", "request.txt", "file to read and parse for request data. read stdin if empty is provided")
	flag.StringVar(&o.output, "output", "response.txt", "file to fill with response data. write to stdout if empty is provided")
	flag.BoolVar(&o.returnCode, "exit_as_return_code", false, "exit code will be the same as return code if true. if false the return code will be the first line of the output")
	flag.BoolVar(&o.verbose, "verbose", false, "output debugging information on standard output")
	flag.StringVar(&o.cipherKey, "cipher_key", "", "cipher key to use for encryption and decryption")
	flag.StringVar(&encryption, "encrypt", "", "encryption algorithms to use as listed on https://8gwifi.org/CipherFunctions.jsp. Only AES/ECB/PKCS5PADDING is supported at the moment.")
	flag.StringVar(&o.encode, "encode", "", "comma separated encoding algorithms to apply. will be applied after encryption. Only base64 and crc32 are supported at the moment")
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
		case "ENCODE":
			o.encode = matches[2]
			break
		case "DECODE":
			o.decode = matches[2]
			break
		case "MODE":
			o.mode = matches[2]
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

func printOptions(o options) {
	if !o.verbose {
		return
	}

	fmt.Printf("Options parsed\n")
	fmt.Printf("----------\n")
	e := reflect.ValueOf(&o).Elem()
	for i := 0; i < e.NumField(); i++ {
		fmt.Printf("%v: %v\n", e.Type().Field(i).Name, e.Field(i))
	}
	fmt.Println()
}

func printInput(verbose bool, input, method, url string, req []byte, header http.Header, extra []string, encoded bool) {
	if !verbose {
		return
	}

	if encoded {
		fmt.Printf("INPUT Data (Encoded)\n")
	} else {
		fmt.Printf("INPUT Data\n")
	}
	fmt.Printf("----------\n")
	fmt.Printf("Request file: %s\n", input)
	fmt.Printf("HTTP Method: %s\n", method)
	fmt.Printf("URL: %s\n", url)
	fmt.Printf("Header: %s\n", header)
	fmt.Printf("Request Body:\n")
	if len(req) > 0 {
		fmt.Printf("%s\n", string(req))
	}

	if len(extra) > 0 {
		fmt.Println()
		fmt.Printf("Extra encoding info\n")
		fmt.Printf("----------\n")
	}
	for _, e := range extra {
		fmt.Println(e)
	}

	fmt.Println()
}

func printOutput(verbose bool, output string, code int, resp []byte, encoded bool) {
	if !verbose {
		return
	}

	if encoded {
		fmt.Printf("OUTPUT Data (Encoded)\n")
	} else {
		fmt.Printf("OUTPUT Data\n")
	}
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
