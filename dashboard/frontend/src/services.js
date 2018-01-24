//let host = "http://localhost:8080";
let host = "";

export function getTFJobListService(namespace) {
  return fetch(`${host}/api/tfjob/${namespace}`).then(r => r.json());
}

export function createTFJobService(spec) {
  let myHeaders = new Headers();
  myHeaders.append("Content-Type", "application/json");
  const options = {
    method: "POST",
    headers: myHeaders,
    body: JSON.stringify(spec)
  };

  return fetch(`${host}/api/tfjob`, options).then(r => r.json());
}

export function getTFJobService(namespace, name) {
  return fetch(`${host}/api/tfjob/${namespace}/${name}`).then(r => r.json());
}

export function deleteTFJob(namespace, name) {
  let myHeaders = new Headers();
  myHeaders.append("Content-Type", "application/json");
  const options = {
    method: "DELETE",
    headers: myHeaders
  };

  return fetch(`${host}/api/tfjob/${namespace}/${name}`, options).then(r =>
    r.json()
  );
}

export function getPodLogs(namespace, name) {
  return fetch(`${host}/api/logs/${namespace}/${name}`).then(r => r.json());
}

export function getNamespaces() {
  return fetch(`${host}/api/namespace`).then(r => r.json());
}
