# cli-to-http

**FAIR WARNING:** This tool was created to help a legacy application make various HTTP requests easily. If you think you need this tool, you're likely wrong. Widely used software like cURL and its successors are much better.

## How-to use it

### Simplest way

1. Put a file with all required data into *request.txt* in the same directory as your [executable](https://github.com/peteraba/cli-to-http/releases/latest).
2. Make sure that the file is properly formatted:
    * First line should contain the URL to be called (unless it's provided as an argument, see below)
    * After the URL there can be a number of lines for headers
    * Finally the content of the data to be sent
3. Your data will be sent via POST.

### Options

*cli-to-http* comes with a number of options, please check them via executing your binary with `--help` argument. Among others you can provide the URL, the headers, the request method and the input and output files as arguments.
