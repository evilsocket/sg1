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

- [x] Working PoC to xfer data in one direction only ( `input channel` -> `module/raw` -> `output channel` ).
- [ ] [**WIP**] Bidirectional communication, aka moving from the concept of `channel` to `tunnel`.
- [ ] SOCKS5 tunnel implementation, once done sg1 can be used for browsing and tunneling arbitrary TCP communications.
- [ ] Deployment with `sg1_deploy` command, with "deploy tunnels" like `-deploy ssh:user:password@host:/path/` (deploy tunnels can be obfuscated as well).
- [ ] Orchestrator to create a randomized and encrypted exfiltration chain of tunnels in a TOR-like network.
- [ ] Profit.

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

## Examples

In the following examples you will always see 127.0.0.1, but that can be any ip, the tool is tunnelling data locally as a PoC but it also works among different computers on any network (as shown by one of the pictures).

--

TLS client -> server session (if no `--tls-pem` or `--tls-key` arguments are specified, a self signed certificate will be automatically generated by sg1):
![tls](https://pbs.twimg.com/media/DPPSi8KW4AIVDVo.jpg:large)

Simple file exfiltration over DNS:
![file](https://pbs.twimg.com/media/DPH8KkAWsAE5rZZ.jpg:large)

Quick and dirty AES encrypted chat over TCP:
![aes-tcp](https://pbs.twimg.com/media/DPHAlOXWAAA9kKv.jpg:large)

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

Just use `sg1 -h` to see a list of available channels and modules, try to pipe them and see what happens, you can also transfer files and make requests "bounce" to several machines with random AES keys ^_^

## License

SG1 was made with ♥  by [Simone Margaritelli](https://www.evilsocket.net/) and it's released under the GPL 3 license.

