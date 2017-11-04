# TODO(jlewi): How can we make it work with golang:1.8.2-alpine
FROM golang:1.8.2

RUN mkdir -p /opt/dashboard
RUN mkdir -p /opt/dashboard/frontend
RUN mkdir -p /opt/dashboard/backend

COPY backend /opt/dashboard/backend/
COPY build /opt/dashboard/frontend/build
RUN chmod a+x /opt/dashboard/backend/backend
WORKDIR /opt/
ENTRYPOINT ["/opt/dashboard/backend/backend"]