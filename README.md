# tcprelay

a simple tcp relay with no runtime depenencies 

## purpose:
I wanted to use socat in a docker container built without the required libraries; this produced no joy.  This is my solution.

## example:
This listens for and accepts connections to port 2222 on your local machine, relaying as connects to port 22 on remotebox
```
tcprelay -lport 2222 -rhost remotebox -rport 22
```

## options:
```
Usage of ./tcprelay:
  -4	specify IPV4
  -6	specify IPV6
  -lhost string
    	local listen host (optional)
  -lport string
    	local listen port
  -rhost string
    	remote host (default "127.0.0.1")
  -rport string
    	remote port
  -verbose
    	output connection state changes
```

## built and tested on: 
```
Linux phobos 4.19.0-21-amd64 #1 SMP Debian 4.19.249-2 (2022-06-30) x86_64 GNU/Linux
```

## install from source:
```
make build && make install
```

## download release:
```
curl -L https://github.com/rstms/tcprelay/releases/download/v0.0.1/tcprelay -o tcprelay
```

## Note
I'm presently a total neophyte in terms of releasing go programs.  If you're reading this, I'd appreciate kindly feedback.
