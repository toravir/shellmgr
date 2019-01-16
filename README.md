# shellmgr                       
Shell Manager manages various shells and provides a REST Interface over unix/tcp socket.

Shell Manager
Use this daemon (process) to spawn/interact with shells

Usage:
```
shellmgr -sockaddr <listenURL> [-socktype <unix|tcp>]
```                                                  

Use `-socktype` to specify type of socket - unix or tcp sockets are supported now (default is unix socket)

Use `-sockaddr` to specify the URL to listen on (mandatory arg)

> for unix sockets, use a path like `/tmp/shellmgr.sock`
>                                                       
>for tcp sockets, use a hostname:port like `localhost:12345`


## REST Interface

Three APIs are supported as of now:
1. `/newShell`
2. `/{shellId}/exec`
3. `/listShells`


### `1. /newShell`

Use this API to create a new Shell - shellMgr will return a shell-Id - using which you can execute commands in the shell.

```
POST /newShell
{
"shell"                 : "/bin/bash"
"commandTimeout"        : 1000,
"terminatePattern"      : "qqqqqq",
"readBufSize"           : 10240
}
```


Parameters for new Shell creation:

`shell` (string) - Executable to run to get the shell running - use absolute path

`commandTimeout` (+ve number) - Max time to wait for the command to complete - this is a default value - can be overridden for each command - timeout is specified in milli
seconds to wait. Default value is 1000 ms.

`terminatePattern` (string) - This is a pattern that shell manager will use to determine if all the output for a command is printed or not - if this pattern is seen, shell
Mgr assumes all the output has been printed out - or else it will wait for the `commandTimeout` - this can be overridden for each command.

`readBufSize` (+ve number) - This parameter sets size of the buffer in bytes used for reads. Too small will result in lots of cpu, too high will result in large memory consumption. Default value is 1024 bytes.

Examples:
```
curl -H 'Content-Type: application/json' http://localhost:12345/newShell \
-d '{"shell":"/bin/bash", "commandTimeout": 1000, "terminatePattern":"qqqq", "readBufSize":10240}'
```

**RESPONSE**
```
{
"shellId":2,
"error":""
}
```
`shellId` (number) - Use this number to identify the particular shell to execute commands and check status

`error` (string) - This will contain any errors encountered when creating the shell


### `2. /{shellId}/exec`

Use this API to request a command to be executed in a particular shell.
```
POST /2/exec
{
"command":"ls -l ; echo -n qqqqq\n",
"commandTimeout" : 1000,
"terminatePattern" : "qqqqqq"
}
```
Parameters:

`command` (string) - Use this parameter to specify the command to be executed in the shell - include any newlines that need to be entered to indicate end of the command. String together multiple commands using whatever delimiter the shell allows - in case of bash - it is `';'`

`commandTimeout` (optional +ve number) - Use this parameter to specify the max time in milliseconds before timing out this command execution - if not specified - the timeout specified from the shellCreation will be used

`terminatePattern` (optional string) - Use this parameter to specify the string pattern that indicates the end of the command output - if not specified will be inherited from the shell creation

Example:
```
curl -H 'Content-Type: application/json' http://localhost:12345/1/exec \
-d '{"command": "ls -l /tmp/ssl_dbg ; echo -n qqqq\n"}'
```


**RESPONSE**
```
{ 
  "output":"-rw-rw-r-- 1 ravir ravir 150163 Jan  7 22:08 /tmp/ssl_dbg\nqqqq",
  "error":""
}
```
`output` (string) - contains the stdout received when executing the command

`error` (string) - contains the stderr received when executing the command - if the command timed-out - it will contain `"%%CMD Timed out !!"`. Timeout can be seen when the terminate pattern is NOT detected - please include any newlines in the terminatePattern

Example with stderr:
```
{
  "output":"qqqq",
  "error":"ls: cannot access '/tmp/non_exist_file': No such file or directory\n"
}
```

### `3. /listShells`

Use this api to get a list of current shells and their statuses including their shellIds.

```
GET /listShells
{
}
```
No Parameters are needed for this request.

Example:
```
curl -X GET -H 'Content-Type: application/json' http://localhost:12345/listShells
```

**RESPONSE**
```
[
   {
      "terminatePattern" : "qqqq",
      "cmdTimeout" : 1000,
      "status" : "running",
      "shell" : "/bin/bash",
      "shellId" : 1,
      "readBufSize": 10240,
      "error" : ""
   },
   {
      "error" : "exec: not started",
      "shellId" : 2,
      "shell" : "/bin/zsh",
      "status" : "Exited",
      "cmdTimeout" : 1000,
      "readBufSize": 1024,
      "terminatePattern" : "qqqq"
   }
]
```

## TODO
In near future these features will be added:
 1. Auto generate terminate Pattern
 2. HTTPS support
 3. Test code
 4. Client code in python/go
 5. UDP Socket ??
 
~~6. Logger~~

~~7. Config to create default set of shells~~ 

 8. Monitor Shells and respawn if needed
 9. Steps to keepalive shells (so that they don't terminate due to inactivity)
10. Create pseudo-TTY for shells (for shells that require a TTY)



