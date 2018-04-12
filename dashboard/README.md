# Kubeflow's Dashboard for TfJobs

## Developer Guide

### Dependencies
* [Nodejs](https://nodejs.org/en/)
* [Yarn](https://yarnpkg.com/en/docs/install)

Then install the dependencies of the frontend React application with:
```sh
cd dashboard/frontend
yarn install
```

### Starting the dashboard locally

First we need to start the backend server:

```sh
export KUBECONFIG=$(echo ~/.kube/config)
go run dashboard/backend/main.go
```

At this point the backend will be running on port `8080`.

Start the frontend's development server in another terminal with:
```sh
cd dashboard/frontend
yarn start
```

This should open a window in your browser (in port `3000`).  

The development server will rebuild the application and refresh your browser automatically every time you save a file.





