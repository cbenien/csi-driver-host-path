FROM ubuntu
#LABEL maintainers="Kubernetes Authors"
LABEL description="AzureBlob Driver"

RUN apt update && apt install -y curl zip man fuse
RUN curl https://rclone.org/install.sh | bash
COPY ./bin/azureblobplugin /azureblobplugin
ENTRYPOINT ["/azureblobplugin"]

# docker run -i -t --device /dev/fuse --privileged --rm --entrypoint /bin/bash azureblobplugin:latest
# rclone mount --daemon --azureblob-account cbenienblobtest --azureblob-key a2pZKZMswNputBuzuYFvUxjw06rX/EcIeDNqdKIs2FEFW/DqrDc6SpOsySnPaskdXJacr18n7KQToNgaWXeAkQ== --vfs-cache-mode writes :azureblob:test/vol1 /blobby