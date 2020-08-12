# Overview

Developing applications with go and docker are pretty easy, but some of routine operations happen too many times, 
like compile, deploy check. etc and most of projects had some kind of machinery like make or custom solutions.
 
This tool `D-go` (aka docker go) is intended to help with this approach and provide smooth and fast experience.

dgo had following conceptual features:

1. `cross compile` locally to perform smooth compile experience
    * Allow to have local replace overrided in go.mod file.
    * Docker just combine all stuff with dependencies, no need to compile inside docker. 
2.  `debug` tests with smooth experience.

3. Both local and docker scenarios looks same.

# Try it now.

## Installation

    go get -u github.com/haiodo/dgo

## Initialize new project
To start working with `dgo init` and it will create a basic Dockerfile with tool inside to compile and test application. 

## Use with existing project

Just call  `dgo build` it will find all applications inside root and cross compile them for x86_64 docker linux.
It will output all binaries into local ./dist folder. So they could be easy compied into docker container.  

# dgo usage scenarios.

# Local scenarios    

1.1 `nsm build`  - just build all stuff and docker conatiner
    
1.2 `nsm test` - perform a build and run tests inside docker.
        
1.2.1 Debug of tests
            `nsm test --debug`, will run tests with dlv to debug contaniner
           
1.2.2 Debug of selected test
            `nsm test --debug --test nsmgr-test.test` - will run debug only for one package, will filter other packages.

# Docker scenarios

### 1. All inside docker
    
2.1 `docker build .` - perform build inside docker, no local SDK references are allowed.
        Inside: nsm build
    
2.2 `docker run $(docker build -q . --target test) - perform execution of tests.
        Inside: nsm build
                nsm test (will check if inside docker will just run all tests found in /bin/*.test)
    
2.3 `docker build . --build-arg BUILD=false` - build container, but copy binaries from local build ./dist folder.
        - require local compile of nsmgr with `nsm build` or
        Inside:
            docker copy 
        
2.4 `docker run $(docker build -q . --target test --build-arg BUILD=false)` - perform execution of tests, copy test binaries from local host.
        Inside:
            docker copy
            nsm test (will check if inside docker will just run all tests found in /bin/*.test)
                will start spire server and run all tests
2.5 Debug container inside docker

# Spire setup.

1.1 Install spire/spiffie locally 

1.2   start dgo with spire
    `dgo spire --root={path}/spire_root`

1.3 add environment variable: SPIFFE_ENDPOINT_SOCKET to IDE.

    `SPIFFE_ENDPOINT_SOCKET=unix:/{path}/spire_root/agent.sock`
