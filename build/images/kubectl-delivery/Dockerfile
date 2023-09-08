FROM alpine:3.17 AS build

# Install kubectl.
ENV K8S_VERSION v1.27.0
RUN apk add --no-cache wget
RUN wget -q https://storage.googleapis.com/kubernetes-release/release/${K8S_VERSION}/bin/linux/amd64/kubectl
RUN chmod +x ./kubectl
RUN mv ./kubectl /bin/kubectl

FROM alpine:3.17
COPY --from=build /bin/kubectl /bin/kubectl
RUN apk add --no-cache bash
COPY scripts/kubectl-delivery/deliver-kubectl.sh .
ENTRYPOINT ["./deliver_kubectl.sh"]
