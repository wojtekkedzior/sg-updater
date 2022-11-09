##/bin/bash

rm sgupdater.zip 
zip sgupdater.zip sgupdater 
aws --profile wpuser --region eu-west-1 lambda update-function-code --function-name SGupdater --zip-file fileb://sgupdater.zip
