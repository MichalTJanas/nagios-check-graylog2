nagios-check-graylog2
===

Nagios Graylog2 checks via REST API the availability of the service. 

- Is the service processing data?
- How long does the check take?
- Monitoring performance
  - through the number of data sources,
  - total processed messages, 
  - index failures
  - and the actual throughput.

This plugin is written in standard Go which means there are no third party libraries used and it is plattform independant. It can compile on all available Go architectures and operating systems (Linux, *BSD, Mac OS X, Windows, ...).

## Installation: 

Just download the source and build it yourself using the go-tools.

    $ go get github.com/catinello/nagios-check-graylog2
    $ mv $GOPATH/bin/nagios-check-graylog2 check_graylog2

## Development
* clone the repo
* modify
* run & build

    
````
go run main.go ... PARAMS
go build -o check_graylog
for linux = GOOS=linux GOARCH=amd64 go build -o check_graylog main.go 
````

## Usage:

    -c string
    	Index error critical limit. (optional)
    -ibc string
    	Input buffer rate below critical threshold in events/second. (optional)
    -l string
    	Graylog API URL (default "http://localhost:12900")
    -p string
    	API password
    -pbtc string
    	Process buffer Time critical threshold in s. (optional)
    -u string
    	API username
    -uc string
    	Uncommited journal entries critical threshold. (optional)
    -version
    	Display version and license information. (info)
    -w string
    	Index error warning limit. (optional)

## Debugging:

Please try your command with the environment variable set as `NCG2=debug` or prefixing your command for example on linux like this.

    NCG2=debug /usr/local/nagios/libexec/check_graylog2 -l http://localhost:9000/api/ -u USERNAME -p PASSWORD -w 10 -c 20

## Examples:

    $ ./check_graylog2 -l http://localhost:12900 -u USERNAME -p PASSWORD -w 10 -c 20
    OK - Service is running!
    768764376 total events processed
    0 index failures
    297 throughput
    1 sources
    Check took 94ms
    |time=0.0094;;;; total=768764376;;;; sources=1;;;; throughput=297;;;; index_failures=0;;;;

    $ ./check_graylog2 -l http://localhost:12900 -u USERNAME -p PASSWORD -w 10 -c 20
    CRITICAL - Can not connect to Graylog2 API|time=0.000000;;;; total=0;;;; sources=0;;;; throughput=0;;;; index_failures=0;;;;

    $ ./check_graylog2 -l https://localhost -insecure -u USERNAME -p PASSWORD -w 10 -c 20
    UNKNOWN - Port number is missing. Try https://hostname:port|time=0.000000;;;; total=0;;;; sources=0;;;; throughput=0;;;; index_failures=0;;;;
    
     $ ./check_graylog2 -l http://localhost:12900 -u USERNAME -p PASSWORD -w 10 -c 20
    CRITICAL - Index Failure above Critical Limit!
    Service is running
    533732628 total events processed
    21 index failures
    297 throughput
    1 sources
    Check took 94ms
    |time=0.0094;;;; total=533732628;;;; sources=1;;;; throughput=297;;;; index_failures=21;;;;


## Return Values:

Nagios return codes are used.

    0 = OK
    1 = WARNING
    2 = CRITICAL
    3 = UNKNOWN

## Icinga2 integration
```
object CheckCommand "check_graylog" {
        import "plugin-check-command"
        command = [ PluginDir + "/check_graylog"]
        arguments = {
            "-l" = {
                  value       = "$url$"
                  description = "Graylog URL"
		      required    = true
            }
            "-p" = {
		      value 	= "$password$"
                  description = "Graylog API Password"
		      required    = true
		}
            "-u" = {
		    value 	= "$user$"
		    description = "Graylog API User"
		    required    = true
		}
            "-uc" = {
		      value 	= "$uncommited$"
                  description = "Threshold for uncommited journal entries"
		      set_if 	= {{ var name = macro("$uncommited$"); return typeof(name) == Number }}
		}
            "-ibc" = {
		    value 	= "$input_buffer$"
		    description = "Lower threshold for input buffer rate"
		    set_if 	= {{ var name = macro("$input_buffer$"); return typeof(name) == Number }}
		}
            "-pbtc" = {
		    value 	= "$process_buffer_time$"
		    description = "Threshold for process buffer time"
		    set_if 	= {{ var name = macro("$process_buffer_time$"); return typeof(name) == Number }}
            }
		"-w" = {
		    value 	= "$index_warning$"
		    description = "Index error warning limit"
		    set_if 	= {{ var name = macro("$index_warning$"); return typeof(name) == Number }}
            }
		"-c" = {
		    value 	= "$index_critical$"
		    description = "Index error critical limit"
		    set_if 	= {{ var name = macro("$index_critical$"); return typeof(name) == Number }}
            }
	}
}
```
## License:

&copy; [Antonino Catinello][HOME] - [BSD-License][BSD]

[BSD]:https://github.com/catinello/nagios-check-graylog2/blob/master/LICENSE
[HOME]:https://antonino.catinello.eu

