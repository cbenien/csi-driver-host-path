FROM ubuntu
#LABEL maintainers="Kubernetes Authors"
LABEL description="AzureBlob Driver"

RUN apt update && apt install -y curl zip man
RUN curl https://rclone.org/install.sh | bash
COPY ./bin/azureblobplugin /azureblobplugin
ENTRYPOINT ["/azureblobplugin"]
