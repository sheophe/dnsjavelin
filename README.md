# dnsjavelin — stress tester for DNS servers using NXDOMAIN attack

### Build

```
go build
```

### Run

```
dnsjavelin -d <domain_name> -n <n_of_threads> -c <n_of_questions>
```

By default the only mandatory parameter is `-d` — domain name. The program will
automatically resolve the domain, obtain all avaliable DNS servers for it, and then
run the attack on each server.