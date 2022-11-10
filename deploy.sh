##/bin/bash

rm sgupdater
rm sgupdater.zip 
go build
zip sgupdater.zip sgupdater 
aws --profile wpuser --region eu-west-1 lambda update-function-code --function-name SGupdater --zip-file fileb://sgupdater.zip
