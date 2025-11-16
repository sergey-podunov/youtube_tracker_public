#!/usr/bin/env sh

kubectl patch cronjob statistics-scheduler -n default -p '{"spec":{"suspend":true}}'
