# cli-to-http

**FAIR WARNING:** This tool was created to help a legacy application make various HTTP requests easily. If you think you need this tool, you're likely wrong. Widely used software like cURL and its successors are much better.

## How-to use it

Please note that *cli-to-http* is only tested sending XML payloads but should work with anything as long as your payload does not start with lines that could be understood as directives. If that's a potential case, just add an empty line before your payload.

### Writing everything into the request file

1. Put a file with all required data into `request.txt` in the same directory as your [executable](https://github.com/peteraba/cli-to-http/releases/latest).
2. Make sure that the file is properly formatted as seen in the [example request file](https://github.com/peteraba/cli-to-http/blob/master/request.txt).
3. Run the *cli-to-http* (or *go run main.go*)
4. Read the response from `response.txt`. [An example is provided](https://github.com/peteraba/cli-to-http/blob/master/response.txt).

### Options

*cli-to-http* comes with a number of options, please check them via executing your binary with `--help` argument. Among others you can provide the URL, the headers, the request method and the input and output files as arguments.

Using the options it is possible to only provide the payload in the input file and have the response payload in the output file.

### Reading stdin and writing stdout

 *cli-to-http* reads payload (and directives) from `request.txt` and writes response data to `response.txt` by default. However if you provde an empty string as input or output, stdin and stdout will be used, respectively.