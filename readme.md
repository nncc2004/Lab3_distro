De momento esta es la estructura

C:.
│   .gitignore
│   2026_02_Lab3_v4.pdf
│   Tutorial_VMs.pdf
│
├───proto
│       placeholder
│
├───VM1
│   │   docker-compose.yml
│   │
│   ├───distros
│   │   │   Dockerfile
│   │   │   main.go
│   │   │
│   │   └───csv
│   │           opcion-1.csv
│   │           opcion-2.csv
│   │           opcion-3.csv
│   │
│   └───mysql
│           init.sql
│
├───VM2
│   │   docker-compose.yml
│   │
│   ├───agente
│   │       Dockerfile
│   │       main.go
│   │
│   ├───procesadora
│   │       Dockerfile
│   │       main.go
│   │
│   └───rabbitmq
│           placeholder
│
├───VM3
│   │   docker-compose.yml
│   │
│   ├───nameserver
│   │       Dockerfile
│   │       main.go
│   │
│   └───procesadora
│           Dockerfile
│           main.go
│
└───VM4
    │   docker-compose.yml
    │
    ├───cuartel
    │       Dockerfile
    │       main.go
    │
    ├───policia
    │       Dockerfile
    │       main.go
    │
    └───procesadora
            Dockerfile
            main.go