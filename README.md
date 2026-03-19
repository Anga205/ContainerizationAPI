# ContainerizationAPI

The general idea of this project is to provide a service similar to the [CodeExecutionAPI](https://github.com/thealcodingclub/CodeExecutionAPI) by [The Alcoding Club](github.com/thealcodingclub) but to manually implement the containerization using cgroups, namespaces, chroot and a bunch of other unix/linux utility tools.

I'm making this as part of a bigger project to host my own coding contests on a platform a little better than hackerrank. The Alcoding Club's API is fine but I need one thats more efficient and has less overhead than firejail.

## Requirements

To use this API, there aren't really any requirements, you just need to be able to send HTTP requests
To **Host** this API, you need to have Linux or WSL (if you're on windows) or MacOS, because this project makes heavy use of unix-specific tools.

## Usage

Just send an HTTP request to the `/execute` endpoint (sample requests with sample curl commands given below), 
If you've already used [CodeExecutionAPI](https://github.com/thealcodingclub/CodeExecutionAPI), this is basically a drop-in replacement for that project

### Route: /execute

This route takes 4 fields:

1. `langauge`: The language of the code snippet.

<details>
    <summary>Click to see supported languages</summary>

    - python
    - rust
    - cpp
    - c
    - java

</details>

2. `code`: The code snippet to be executed.
3. `timeout`: The maximum time in seconds for which the code should run. If the code runs for more than this time, it will be terminated. 
    - default: **5 seconds**
    - max: **60 seconds**
4. `max_memory`: The maximum memory in KB (kilobytes) that the code can use. If your code tries to use more memory than this, it'll encounter a memory limit error.
    - default: **32768KB** (or 32MB)
    - max: **131072KB** (or 128MB)

**Note:** The default and maximum values apply to the publically hosted URL of this repository, but if you are hosting your own instance, you can change the default and max values with environment variables (mentioned at the end of this README)

---

### Example Requests

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
curl --location 'https://codeapi.anga.codes/execute' \
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
    "cpu_time": "125.034027ms"  // in Seconds
}
```

---

#### Request body format (Example 2):

```json
{
    "language": "python",
    "code": "import time\nprint('Hello World')\ntime.sleep(5)",
    "timeout": 2,       // in seconds (defaults to 5, max 60)
}
```

<details>
<summary>Click to copy curl command</summary>

```bash
curl --location 'https://codeapi.anga.codes/execute' \
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
    "cpu_time": "2.000637132s"      // Time before code was terminated
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
curl --location 'https://codeapi.anga.codes/execute' \
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
    "error": "Traceback (most recent call last):\n  File \"<string>\", line 1, in <module>\n  File \"<string>\", line 1, in <listcomp>\nMemoryError\n",
    "memory_used": "283964 KB",
    "cpu_time": "803.820445ms"
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
curl --location 'https://codeapi.anga.codes/execute' \
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
    "memory_used": "13400 KB",
    "cpu_time": "135.266682ms"
}
```

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