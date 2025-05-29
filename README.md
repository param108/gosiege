This document provides information about the Siege command-line tool, including how to build, use, and configure it.

# Overview
Siege is a command-line tool designed to perform load testing by simulating HTTP requests. It allows users to specify configurations such as maximum requests per second, maximum concurrent requests, and test duration.

## Building the Project
To build the Siege tool, ensure you have Go installed on your system. Follow these steps:

```
## Clone the repository
git clone https://github.com/param108/siege.git

# Navigate to the project directory
cd siege

# Build the project
make build
```

After building, the executable ~siege~ will be available in the project directory.

## Command Options
The Siege tool (`sg`) provides the following commands and options:

+ **Command: run**
  - Runs the siege test with the provided configuration.

+ **Options:**
  1. ~--config~ or ~-c~: Path to the configuration file (mandatory).
  2. ~--max-rps~ or ~-r~: Maximum requests per second (optional).
  3. ~--max-concurrent~ or ~-m~: Maximum number of concurrent requests (optional).
  4. ~--duration~ or ~-d~: Duration of the siege in seconds (default: 60 seconds).

## Usage Examples
Below are examples of how to use the Siege tool:

+ Run a siege test with a configuration file:
``` bash
./sg run -c /path/to/config.json
```

+ Run a siege test with additional options:
The commandline parameters override those in the config file.
```
./sg run -c /path/to/config.json -r 100 -m 50 -d 120
```

## Configuration File
The configuration file specifies the parameters for the siege test. It should be in JSON format. An example file is provided in `sample_config.json` Example:

```json
{
  "urls": [
    {
      "url": "http://example.com/api/resource",
      "method": "GET",
      "headers": {
        "Authorization": "Bearer randomtoken123",
        "Accept": "application/json"
      },
      "body": "",
      "repeat": 5
    },
    {
      "url": "http://test.com/api/data",
      "method": "POST",
      "headers": {
        "Content-Type": "application/json"
      },
      "body": "{\"key\": \"value\"}",
      "repeat": 3
    },
    {
      "url": "http://randomsite.org/api/submit",
      "method": "PUT",
      "headers": {
        "User-Agent": "SiegeClient/1.0"
      },
      "body": "{\"update\": \"true\"}",
      "repeat": 2
    }
  ],
  "duration": 45,
  "max_concurrent": 100,
  "max_rps": 200
}
```

## Output
The Siege tool provides detailed statistics after the test. Example output:

```
Final Stats:
Max RPS: 100.00
Current Requests: 75.00
Max Concurrents: 50
2xx Responses: 1200
4xx Responses: 50
5xx Responses: 10
Connection Failures: 5
```

## Handling Interrupts
Siege gracefully handles interrupts (e.g., Ctrl+C) by canceling the test and providing the latest statistics.

## License
This project is licensed under the MIT License. See the LICENSE file for details.

## Contributing
Contributions are welcome! Please submit issues or pull requests to the repository.

## Contact
For questions or support, contact the repository owner or submit an issue on GitHub.
