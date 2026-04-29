#!/bin/bash
# build the golang binary for linux
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o rmstool
chmod +x rmstool
# copy the binary to the scripts folder for inclusion in the image
cp rmstool isobuild

# build the docker image
cd isobuild
docker build --platform linux/amd64 -t rmshelpervm:kvm .
docker run --platform linux/amd64 -v $(pwd)/iso:/iso rmshelpervm:kvm

# deploy the image to morpheus
cd ..
cp isobuild/iso/alpine-helper_kvm-v3.22-x86_64.iso citesting/vme
cd citesting/vme
# run the vme tests
go run main.go


#sh aports/scripts/mkimage.sh --tag v3.22 --arch x86_64 --outdir /iso --repository http://10.0.0.85/v3.22/main --repository http://1
#0.0.0.85/v3.22/community --profile helper_kvm