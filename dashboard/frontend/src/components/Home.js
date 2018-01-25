import React, { Component } from "react";
import { Route, Switch } from "react-router-dom";
import SelectField from "material-ui/SelectField";
import MenuItem from "material-ui/MenuItem";

import "./App.css";
import JobList from "./JobList";
import Job from "./Job";
import CreateJob from "./CreateJob";
import AppBar from "./AppBar";
import { getTFJobListService, getNamespaces } from "../services";

const allNamespacesKey = "All namespaces";

class Home extends Component {
  constructor(props) {
    super(props);
    this.state = {
      tfJobs: [],
      selectedNamespace: allNamespacesKey,
      namespaces: []
    };

    this.handleNamespaceChange = this.handleNamespaceChange.bind(this);
    this.lastNamespaceQueried = allNamespacesKey;
  }

  componentDidMount() {
    this.fetchJobs();
    this.fetchNamespaces();
    setInterval(() => this.fetchJobs(), 10000);
  }

  fetchJobs() {
    let ns =
      this.state.selectedNamespace === allNamespacesKey
        ? ""
        : this.state.selectedNamespace;
    getTFJobListService(ns)
      .then(b => {
        this.lastNamespaceQueried = this.state.selectedNamespace;
        this.setState({ tfJobs: b.items });
      })
      .catch(console.error);
  }

  fetchNamespaces() {
    getNamespaces()
      .then(b =>
        this.setState({
          namespaces: b.items
            .map(ns => ns.metadata.name)
            .concat(allNamespacesKey)
        })
      )
      .catch(console.error);
  }

  handleNamespaceChange(event, index, value) {
    this.setState({ selectedNamespace: value });
  }

  render() {
    if (this.lastNamespaceQueried !== this.state.selectedNamespace) {
      // if the user changed the selected namespace we want to refresh immediatly, not once the timer ticks.
      this.fetchJobs();
    }

    const nsl = this.state.namespaces.map(ns => {
      return <MenuItem value={ns} primaryText={ns} key={ns} />;
    });

    return (
      <div>
        <AppBar />
        <div id="main" style={this.styles.mainStyle}>
          <div style={this.styles.list}>
            <SelectField
              floatingLabelText="Namespace"
              value={this.state.selectedNamespace}
              onChange={this.handleNamespaceChange}
            >
              {nsl}
            </SelectField>
            <JobList jobs={this.state.tfJobs} />
          </div>
          <div style={this.styles.content}>
            <Switch>
              <Route path="/new" component={CreateJob} />
              <Route
                path="/:namespace/:name"
                render={props => <Job jobs={this.state.tfJobs} {...props} />}
              />
              <Route
                path="/"
                render={props => <Job jobs={this.state.tfJobs} {...props} />}
              />
            </Switch>
          </div>
        </div>
      </div>
    );
  }

  styles = {
    mainStyle: {
      minHeight: "800px",
      margin: 0,
      padding: 0,
      display: "flex",
      flexFlow: "row"
    },
    content: {
      margin: "4px",
      padding: "5px",
      flex: "3 1 80%",
      order: 2
    },
    list: {
      margin: "4px",
      padding: "5px",
      flex: "1 6 20%",
      order: 1
    }
  };
}

export default Home;
