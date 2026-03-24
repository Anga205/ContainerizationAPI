# ContainerizationAPI

The general idea of this project is to provide a service similar to the [CodeExecutionAPI](https://github.com/thealcodingclub/CodeExecutionAPI) by [The Alcoding Club](github.com/thealcodingclub) but to manually implement the containerization using cgroups, namespaces, chroot and a bunch of other unix/linux utility tools. This will help accomplish faster execution times and cut down on sandboxing overhead

I'm making this as part of a bigger project to host my own coding contests on a platform a little better than hackerrank. The Alcoding Club's API is fine but I need one thats more efficient and has less overhead than firejail.

## Requirements

To use this API, there aren't really any requirements, you just need to be able to send HTTP requests
To **Host** this API, you need to have Linux or WSL (if you're on windows) or MacOS, because this project makes heavy use of unix-specific tools.

## Usage

Using this API is designed to be as simple as possible, at it's core, it's just an HTTP request that you send to `/execute`, or `/simple-execute`

The following are the routes that this API makes available, click one to see it's documentation

<details>
<summary><h3>POST Route: /simple-execute (click to expand)</h3></summary>

This route is a drop in replacement for [CodeExecutionAPI](https://github.com/thealcodingclub/CodeExecutionAPI), just take the code where you called the old API and replace `https://codeapi.anga.codes/execute` with `${url}/simple-execute` and watch as ur response times get faster!.

This route takes 4 fields:

1. `langauge`: The language of the code snippet.
    <details>
    <summary>Click to see supported languages</summary>

    - python
    - cpp
    - c
    - java

    </details>

2. `code`: The code snippet to be executed.
3. `timeout` (optional field): The maximum number of seconds for which the code should run. If the code takes more time to execute than what it was allocated, it will be terminated.
    - default (value assumed if you dont specify): **5 seconds**
    - max (you can't allocate more seconds than this): **60 seconds**
4. `max_memory` (optional field): The maximum memory in KB (kilobytes) that the code can use. If the code uses more memory than this, it will be terminated.
    - default (value assumed if you dont specify): **32768KB** (or 32MB)
    - max (you can't allocate more RAM than this): **131072KB** (or 128MB)
5. `inputs` (optional field): This is an array of strings where every element is 1 line, its completely optional and you should use it if your code reads input from STDIN (like with python input() or scanf() in C), read example 4 to see usage

*Note: the above mentioned default and max values can be modified by editing the environment variables mentioned at the bottom of this README*

---

#### Request body format (Example 1):

```json
{
    "language": "python",
    "code": "print('Hello World')"
}
```
<details>
<summary>Click to copy curl command</summary>

```bash
curl --location '${url}/simple-execute' \
--header 'Content-Type: application/json' \
--data '{
    "language": "python",
    "code": "print('\''Hello World'\'')"
}'
```

</details>

#### Response body format (Example 1):

```json
{
    "output": "Hello World\n",  // Output of the code
    "error": "",                // If any error occurs during execution
    "memory_used": "13808 KB",  // RAM used (in KB)
    "cpu_time": "9.166097ms"    // as in 9.16 milliseconds
}
```

---

#### Request body format (Example 2):

```json
{
    "language": "python",
    "code": "import time\nprint('Hello World')\ntime.sleep(5)",
    "timeout": 2       // in seconds (defaults to 5, max 60)
}
```

<details>
<summary>Click to copy curl command</summary>

```bash
curl --location '${url}/simple-execute' \
--header 'Content-Type: application/json' \
--data '{
    "language": "python",
    "code": "import time\nprint('\''Hello World'\'')\ntime.sleep(5)",
    "timeout": 2       
}'
```

</details>

#### Response body format (Example 2):

```json
{
    "output": "",                   // No output is returned on timeout
    "error": "Execution Timed Out", // Error message in case of timeout
    "memory_used": "15856 KB",      // RAM used (in KB)
    "cpu_time": "2.00014463s"      // Time before code was terminated
}
```

---

#### Request body format (Example 3):

```json
{
    "language": "python",
    "code": "import random;[random.random() for x in range(10**7)]",
    "max_memory": 300000        // in KB (defaults to 32768, max 131072)
}
```

<details>
<summary>Click to copy curl command</summary>

```bash
curl --location '${url}/simple-execute' \
--header 'Content-Type: application/json' \
--data '{
    "language": "python",
    "code": "import random;[random.random() for x in range(10**7)]",
    "max_memory": 300000
}'
```

</details>

#### Response body format (Example 3):

```json
{
    "output": "",
    "error": "Memory limit exceeded",   // code is terminated the moment it takes more memory than allocated
    "memory_used": "136636 KB",
    "cpu_time": "115.779962ms"
}
```

---

#### Request body format (Example 4):

```json
{
    "language": "python",
    "code": "a = input()\nprint(f'first value entered is {a}.')\nb=input()\nprint(f'second value entered is {b}.')",
    "inputs": [
        "bob",
        "alice"
    ]
}
```

<details>
<summary>Click to copy curl command</summary>

```bash
curl --location '${url}/simple-execute' \
--header 'Content-Type: application/json' \
--data '{
    "language": "python",
    "code": "a = input()\nprint(f'\''first value entered is {a}.'\'')\nb=input()\nprint(f'\''second value entered is {b}.'\'')",
    "inputs": [
        "bob",
        "alice"
    ]
}'
```

</details>

#### Response body format (Example 4):

```json
{
    "output": "first value entered is bob.\nsecond value entered is alice.\n",
    "error": "",
    "memory_used": "24504 KB",
    "cpu_time": "13.625964ms"
}
```


</details>

**Note**: the route `/simple-execute` also calls `/execute` under the hood, its just that the formatting for the input and output are slightly more easy to read and understand. 
<details>
    <summary><h3>POST Route: /execute</h3></summary>

This route takes 4 fields:

1. `langauge`: The language of the code snippet.
    <details>
    <summary>Click to see supported languages</summary>

    - python
    - cpp
    - c
    - java

    </details>

2. `code`: The code snippet to be executed.
3. `timeout` (optional field): The maximum number of **nanoseconds** for which the code should run. If the code takes more time to execute than what it was allocated, it will be terminated.
    - default (value assumed if you dont specify): **5 seconds**
    - max (you can't allocate more seconds than this): **60 seconds**
4. `max_memory` (optional field): The maximum memory in KB (kilobytes) that the code can use. If the code uses more memory than this, it will be terminated.
    - default (value assumed if you dont specify): **32768KB** (or 32MB)
    - max (you can't allocate more RAM than this): **131072KB** (or 128MB)
5. `inputs` (optional field): This is a single string that is fed to the code via STDIN at runtime, if you want multiple lines in STDIN then separate them with `\n`, its completely optional and you should use it if your code reads input from STDIN (like with python input() or scanf() in C), read example 4 to see usage

*Note: the above mentioned default and max values can be modified by editing the environment variables mentioned at the bottom of this README*

---

#### Request body format (Example 1):

```json
{
    "language": "python",
    "code": "print('Hello World')"
}
```
<details>
<summary>Click to copy curl command</summary>

```bash
curl --location '${url}/execute' \
--header 'Content-Type: application/json' \
--data '{
    "language": "python",
    "code": "print('\''Hello World'\'')"
}'
```

</details>

#### Response body format (Example 1):

```json
{
    "output": "Hello World\n",  // output from STDOUT
    "error": "",                // output from STDERR
    "cpu_time": 10283909,       // run time in nanoseconds (10ms or 0.01 seconds)
    "memory_used": 25116        // RAM used (in KB)
}
```

---

#### Request body format (Example 2):

```json
{
    "language": "python",
    "code": "import time\nprint('Hello World')\ntime.sleep(5)",
    "timeout": 2000000000       // in nanoseconds (defaults to 5000000000, max 60000000000)
}
```

<details>
<summary>Click to copy curl command</summary>

```bash
curl --location '${url}/execute' \
--header 'Content-Type: application/json' \
--data '{
    "language": "python",
    "code": "import time\nprint('\''Hello World'\'')\ntime.sleep(5)",
    "timeout": 2000000000
}'
```

</details>

#### Response body format (Example 2):

```json
{
    "output": "",                   // No output is returned on timeout
    "error": "Execution Timed Out", // Error message in case of timeout
    "cpu_time": 2000003280,         // Time (in nanoseconds) before code was terminated
    "memory_used": 15856,           // RAM used (in KB)
}
```

---

#### Request body format (Example 3):

```json
{
    "language": "python",
    "code": "import random;[random.random() for x in range(10**7)]",
    "max_memory": 300000        // in KB (defaults to 32768, max 131072)
}
```

<details>
<summary>Click to copy curl command</summary>

```bash
curl --location '${url}/execute' \
--header 'Content-Type: application/json' \
--data '{
    "language": "python",
    "code": "import random;[random.random() for x in range(10**7)]",
    "max_memory": 300000
}'
```

</details>

#### Response body format (Example 3):

```json
{
    "output": "",
    "error": "Memory limit exceeded",   // code is terminated the moment it takes more memory than allocated
    "cpu_time": 116936122,
    "memory_used": 136628
}
```

---

#### Request body format (Example 4):

```json
{
    "language": "python",
    "code": "a = input()\nprint(f'first value entered is {a}.')\nb=input()\nprint(f'second value entered is {b}.')",
    "inputs": "bob\nalice"
}
```

<details>
<summary>Click to copy curl command</summary>

```bash
curl --location '${url}/execute' \
--header 'Content-Type: application/json' \
--data '{
    "language": "python",
    "code": "a = input()\nprint(f'\''first value entered is {a}.'\'')\nb=input()\nprint(f'\''second value entered is {b}.'\'')",
    "inputs": "bob\nalice"
}'
```

</details>

#### Response body format (Example 4):

```json
{
    "output": "first value entered is bob.\nsecond value entered is alice.\n",
    "error": "",
    "cpu_time": 9058716,
    "memory_used": 25136
}
```

</details>

## Environment Variables

- `MAX_TIMEOUT`: This is the maximum timeout that a request can set, if an incoming request has a higher timeout than the value you set here, then it will ignore the request's set timeout only be executed for `MAX_TIMEOUT` time.
    - this env variable uses the **Go Duration String** format e.g - `1m30s`, `4h15s`, `5s`
    - if you dont set this env variable, it defaults to `60s`
    - the public version of this api has this value set to `60s`
- `MAX_MEMORY_LIMIT`: Maximum RAM allocation that a request can be set to, if an incoming request tries to set to a higher memory limit, it will be ignored and reset to this value
    - this must be set to an unsigned integer (positive number) and is measured in Kilobytes (KB)
    - if you dont set this env variable, it defaults to `131072` which is `128 MB` or `131072 KB`
    - the public version of this api has this value set to `131072`
- `DEFAULT_TIMEOUT`: If an incoming request does not have a `timeout` parameter (i.e, request does not specify a time limit) then the sandbox will set the time limit to this value
    - same format as `MAX_TIMEOUT`
    - if you dont set this env variable, it defaults to `5s`
    - the public version of this api has this value set to `5s`
- `DEFAULT_MEMORY_LIMIT`: If an incoming request does not have a `max_memory` parameter (i.e, request does not specify a memory limit) then the sandbox will set the maximum memory limit to this value
    - same format as `MAX_MEMORY_LIMIT`
    - if you dont set this env variable, it defaults to `32 MB`
    - the public version of this api has this value set to `32 MB`
- `RAM_LIMIT`: The total amount of RAM that is available globally to the server, cumulatively, the API will not allocate more RAM than what you define here, for example, if RAM limit if 1GB, and there are already 10 sandboxes that have reserved 100MB (either by default or by user-specification in the `max_memory` parameter) and an 11th process tries to reserve even more RAM, the server will NOT allocate that RAM
    - same format as `MAX_MEMORY_LIMIT`
    - default value is `1048576` or `1 GB`
    - public version of this API sets this to `6 GB`
- `ENABLE_QUEUE`: if false, the server will reject requests when ram limit is reached, if true, the server will queue requests until sufficient ram is available
    - this can only be set to `true` or `false`
    - by default this is `true`
    - the publically deployed API has this set to `true`
- `ENABLE_DEBUG`: if true, the server will expose debug routes (like `/check-ram`) for monitoring and debugging purposes
    - same format as `ENABLE_QUEUE`
    - by default this is `false`
    - the public API has this set to `true`
- `GIN_MODE`: wether to run the gonic-gin server in release mode or debug mode, for this API there wont be much of a performance difference regardless of which one you pick, it is reccomended that you read the docs for [gonic gin](https://gin-gonic.com/en/docs/deployment/#configuration-options) for more information
    - this can be set to `release` or `debug`
    - by default it is set to `debug`
    - the public API has this set to `release`
- `PORT`: Which network port to run the server on
    - this can be set to any unsigned integer value between `0` and `65535`
    - by default this is `8080`
    - public API has this set to `8080`

**Note:** *if you are self-hosting this api, remember to run it with superuser permissions, or a user which has permissions to alter controlgroups and namespaces, normally you can achieve this just by using "sudo" before running the code*