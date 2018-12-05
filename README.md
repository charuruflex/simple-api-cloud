# simple-api-cloud
Hello World API in golang using Kubernetes and configuration for GCP

## Requirements
 - go (tested with go1.11.2)
 - docker (tested with docker 18.03.1-ce)
 - local redis instance for testing (tested with 5.0.2)
 - google-cloud-sdk installed and configured with your credentials
 - a working and configured Kubernetes cluster
 - make
 
## How to start
 - An IP address is required on GCP for Ingress to work: `gcloud compute addresses create simple-api-ip --global`(replace simple-api-ip if needed)
 - In file `k8s/deployment.yml`, replace simple-api-ip with the name you chose on line `kubernetes.io/ingress.global-static-ip-name: simple-api-ip`
 - `make ship` will:
    - get dependencies
    - start tests (a local redis instance is needed to complete tests !!!)
    - build the code
    - build the docker image
    - send docker image on registry
    - launch `k8s/deployment.yml` on the configured Kubernetes cluster

## HTTPS
Follow this guide to use Letsencrypt: https://github.com/ahmetb/gke-letsencrypt

## API
- GET `/`: some informations on the instance
  - `{"app":"simple-api-cloud","hostname":"simple-api-55f76f9647-n8pxf","version":"70ad329"}`
- GET `/hello/Charles`: get birthday message for Charles (number of days till birthday, considering UTC)
    - normal case: `{"message": "Hello Charles! Your birthday is in 300 days"}`
    - when it's Charles' birthday: `{"message": "Hello Charles! Happy birthday!"}`
    - if Charles' birthday is unknown: `{"message":"Hello! Unfortunately I don't know Charles yet. Please add his/her date of birth."}`
    
- PUT `/hello/Charles`: create/update date of birth for Charles
    - req: `{"dateOfBirth": "1988-12-01"}` with date format YYYY-MM-DD

## Deployed version on GCP
A live demo is available here:
- http://revolut-sre.franceskinj.fr
- https://revolut-sre.franceskinj.fr

## TODO
- Use a container for building the API
- Date: a more precise date format can be used: `YYYY-MM-DD hh:mm:ss`. The date will be always considered UTC. If the hour is not mentioned, it will default to 00:00:00 of the given date. The date will be stored as unix timestamp.
- Spanner: more adapted with a financial context. Plus, it's all ready for scalability, availability, performance, multi-regions and totally managed. The only drawback is price...
- Stackdriver: for monitoring the whole stack
- Use Autoscaling with Custom Metrics: adapts automagically if the load goes up.
- Use Spinnaker for continuous delivery
