import React from "react";
import MuiThemeProvider from "material-ui/styles/MuiThemeProvider";
import getMuiTheme from "material-ui/styles/getMuiTheme";
import FlatButton from "material-ui/FlatButton";

// BrowserRouter is a traditional router, If I currently am at the root `/` and I navigate to `/new`
// the resulting route will be: `${baseroute}/new`. However this breaks with ambassador:
// If I am at `/api/v1/namespaces/kubeflow/services/ambassador:80/proxy/tfjobs/ui/` and I navigate to `/new` I still end up on `/new`
// And we also can't just figure out the proxy part of the url and preprend it to each link
// such as: `/api/v1/namespaces/kubeflow/services/ambassador:80/proxy/tfjobs/ui/new` since the go backend
// is configured to only handle `/tfjobs/ui` and not `/api/v1/...`
// So instead we can use the HashRouter, which will only use the part of the url located after a # to infer it's current state
// So `/api/v1/namespaces/kubeflow/services/ambassador:80/proxy/tfjobs/ui/#`
// will become `/api/v1/namespaces/kubeflow/services/ambassador:80/proxy/tfjobs/ui/#/new.`
// This works regardless of how the dashboard is accessed. This is also how the k8s dashboard works
import { HashRouter as Router, Link } from "react-router-dom";
import { orange700, orange400 } from "material-ui/styles/colors";
import ContentAdd from "material-ui/svg-icons/content/add";

import "./App.css";
import Home from "./Home.js";

const headerStyle = {
  display: "flex",
  backgroundColor: "white",
  height: "56px",
  padding: "12px",
  color: "#f57c00",
  justifyContent: "space-between"
};

const brandingStyle = {
  fontSize: "1.5em",
  marginTop: "3px",
  textAlign: "left"
};

const muiTheme = getMuiTheme({
  appBar: {
    height: 56,
    color: orange700
  },
  raisedButton: {
    primaryColor: orange700
  },
  flatButton: {
    primaryTextColor: orange700
  },
  toggle: {
    thumbOnColor: orange700,
    trackOnColor: orange400
  }
});

const App = () => {
  return (
    <Router>
      <MuiThemeProvider muiTheme={muiTheme}>
        <div className="App">
          <header className="App-header">
            <link
              href="https://fonts.googleapis.com/css?family=Roboto"
              rel="stylesheet"
            />
            <link
              rel="stylesheet"
              href="https://maxcdn.bootstrapcdn.com/bootstrap/latest/css/bootstrap.min.css"
            />
            <link
              href="https://fonts.googleapis.com/icon?family=Material+Icons"
              rel="stylesheet"
            />
            <div style={headerStyle}>
              <h1 style={brandingStyle}>kubeflow/tf-operator</h1>
              <FlatButton
                label="Create"
                primary={true}
                icon={<ContentAdd />}
                containerElement={<Link to="/new" />}
              />
            </div>
          </header>
          <div>
            <Home />
          </div>
        </div>
      </MuiThemeProvider>
    </Router>
  );
};

export default App;
