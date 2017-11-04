// let host = "http://localhost:8080";
let host = "";

export function getTfJobListService() {
    return fetch(`${host}/api/tfjob`)
        .then(r => r.json());
}

export function createTfJobService(spec) {
    let myHeaders = new Headers();
    myHeaders.append("Content-Type", "application/json");
    const options = {
        method: "POST",
        headers: myHeaders,
        body: JSON.stringify(spec)
    };

    return fetch(`${host}/api/tfjob`, options)
        .then(r => r.json());
}

export function getTfJobService(namespace, name) {
    return fetch(`${host}/api/tfjob/${namespace}/${name}`)
        .then(r => r.json());
}

export function deleteTfJob(namespace, name) {
    let myHeaders = new Headers();
    myHeaders.append("Content-Type", "application/json");
    const options = {
        method: "DELETE",
        headers: myHeaders
    };

    return fetch(`${host}/api/tfjob/${namespace}/${name}`, options)
        .then(r => r.json());
}