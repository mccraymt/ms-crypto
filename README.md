
  gocode
  gopkgs
  go-outline
  go-symbols
  guru
  gorename
  godef
  goreturns
  golint
  gotests

## ms-crypto
crypto microservice

## set-up DEV environment
1. set env vars  
  export GSON_API_URL_MS_QUOTE_INTENT="https://ms-crypto-dev.qa-elephant.com"  
  export PORT_MS_QUOTE_INTENT=4000  
  export NEW_RELIC_KEY_MS_QUOTE_INTENT=" "  
  export ENVIRONMENT_MS_QUOTE_INTENT=dev  
  export PREFIX_TABLE_NAME_MS_QUOTE_INTENT=true  
  export SSL_CERT_SIMPLE_HTTP_TEXT=" "  
  export DEBUG_MS_QUOTE_INTENT=true  
  export GO15VENDOREXPERIMENT=1
  * **NOTE** make sure you source your env vars for your current terminal session
  * **NOTE**
    * -set GO15VENDOREXPERIMENT=1 //go1.5
    * -set GO15VENDOREXPERIMENT=0 //go1.6
2. install and configure gvm
  * https://github.com/moovweb/gvm
  1. create a folder for your packageset/repository
     * mkdir -p ~/go//src/github.com/mccraymt
     * cd ~/go/ms-crypto/src/github.com/mccraymt
     * git clone github.com/mccraymt/ms-crypto
     * cd ms-crypto
     * git checkout dev
  2. gvm pkgset create ms-crypto
  3. gvm pkgenv ms-crypto
     * change the following
     * export GOPATH; GOPATH="/Users/obie/.gvm/pkgsets/go1.4.2/ms-crypto:$GOPATH"
     * to
     * export GOPATH; GOPATH="/Users/obie/.gvm/pkgsets/go1.4.2/ms-crypto:$GOPATH:$HOME/go/ms-crypto"
     * save changes
  4. gvm pkgset use ms-crypto
3. install godep
  * go get -u github.com/tools/godep
  * godep restore
4. install fresh
  * go get -u github.com/pilu/fresh
5. install godebug
  * go get -u github.com/mailgun/godebug
6. run website (from root directory)
  * open terminal session
  * use gvm package set
    * gvm pkgset use [your package set name]
  * start website
    * fresh

## set-up PROD environment (CentOS/RedHat)
1. ssh into web server (VAPRD-WEBMS-01, VAPRD-WEBMS-02) as root user
2. install git (if not installed already)
   * git version
     * yum install git
2. install gvm
   * https://github.com/moovweb/gvm
     * MUST set go v1.4.2 as default
3. NOTE: following steps apply to systemd
   * https://serversforhackers.com/video/process-monitoring-with-systemd
4. cd to /opt/
   * create a new directory: mkdir ./ms-crypto
   * clone git repository: git clone https://github.com/mccraymt/ms-crypto.git
   * build project: go build
     * creates an executable named 'ms-crypto'
5. copy web-ms-crypto.service file to: /usr/lib/systemd/system/
6. create a non-privileged user to run the service
   * adduser -r -M -s /bin/false elephant-ms-web-user
7. install the systemd service
   * systemctl disable web-ms-crypto; systemctl enable web-ms-crypto.service
8. start web service: service web-ms-crypto stop; service web-ms-crypto start; service web-ms-crypto status
9. NOTE: whenever the web-ms-crypto.service file changes, run: systemctl daemon-reload
10. Troubleshoot if service doesn't start
   * journalctl -alb -u web-ms-crypto 

### godebug web server example
godebug run -instrument=github.com/mccraymt/ms-crypto/app/policycenter,github.com/mccraymt/ms-crypto/app/models,github.com/mccraymt/ms-crypto/app/controllers,github.com/mccraymt/ms-crypto/app/resources,github.com/mccraymt/ms-crypto/app/mappers,github.com/mccraymt/ms-crypto/config main.go

### godebug unit test example
godebug test -instrument=github.com/mccraymt/ms-crypto/app/policycenter,github.com/mccraymt/ms-crypto/app/models,github.com/mccraymt/ms-crypto/app/controllers,github.com/mccraymt/ms-crypto/app/resources,github.com/mccraymt/ms-crypto/app/mappers,github.com/mccraymt/ms-crypto/config ./app/policycenter
