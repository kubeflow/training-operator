import { getHost } from "./utils";

const host = getHost();

export function getTFJobListService(namespace) {
  return fetch(`${host}/api/tfjob/${namespace}`, {
    method: "GET",
    mode: "cors",
    headers: {
      "Content-Type": "application/json"
    },
    credentials: "include",
    redirect: "follow"
  }).then(r => r.json());
}

export function createTFJobService(spec) {
  const options = {
    method: "POST",
    mode: "cors",
    headers: {
      "Content-Type": "application/json"
    },
    credentials: "include",
    redirect: "follow",
    body: JSON.stringify(spec)
  };

  return fetch(`${host}/api/tfjob`, options).then(r => r.json());
}

export function getTFJobService(namespace, name) {
  return fetch(`${host}/api/tfjob/${namespace}/${name}`, {
    method: "GET",
    mode: "cors",
    headers: {
      "Content-Type": "application/json"
    },
    credentials: "include",
    redirect: "follow"
  }).then(r => r.json());
}

export function deleteTFJob(namespace, name) {
  const options = {
    method: "DELETE",
    mode: "cors",
    headers: {
      "Content-Type": "application/json"
    },
    credentials: "include",
    redirect: "follow"
  };

  return fetch(`${host}/api/tfjob/${namespace}/${name}`, options).then(r =>
    r.json()
  );
}

export function getPodLogs(namespace, name) {
  return fetch(`${host}/api/logs/${namespace}/${name}`, {
    method: "GET",
    mode: "cors",
    headers: {
      "Content-Type": "application/json"
    },
    credentials: "include",
    redirect: "follow"
  }).then(r => r.json());
}

export function getNamespaces() {
  return fetch(`${host}/api/namespace`, {
    method: "GET",
    mode: "cors",
    headers: {
      "Content-Type": "application/json"
    },
    credentials: "include",
    redirect: "follow"
  }).then(r => r.json());
}
