##/bin/bash

rm bootstrap
rm sgupdater.zip 
GOARCH=amd64 GOOS=linux go build -o bootstrap main.go
zip sgupdater.zip bootstrap 
aws --profile wpuser --region eu-west-1 lambda update-function-code --function-name SGupdater --zip-file fileb://sgupdater.zip