FROM alpine:3.17 AS build

# Install kubectl.
ENV K8S_VERSION v1.30.5

RUN apk add --no-cache wget
RUN wget -q https://storage.googleapis.com/kubernetes-release/release/${K8S_VERSION}/bin/linux/amd64/kubectl
RUN chmod +x ./kubectl
RUN mv ./kubectl /bin/kubectl

FROM alpine:3.17
COPY --from=build /bin/kubectl /bin/kubectl
RUN apk add --no-cache bash

ENTRYPOINT ["/bin/sh", "-c"]
CMD ["cp /bin/kubectl /opt/kube/kubectl"]
