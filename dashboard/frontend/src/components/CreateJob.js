import React, { Component } from "react";
import PropTypes from "prop-types";

import { Card, CardText, CardActions } from "material-ui/Card";
import TextField from "material-ui/TextField";
import { withRouter } from "react-router-dom";
import FlatButton from "material-ui/FlatButton";
import ContentAdd from "material-ui/svg-icons/content/add";
import RaisedButton from "material-ui/RaisedButton";
import Divider from "material-ui/Divider";

import { createTFJobService } from "../services";
import RequiredTextField from "./RequiredTextField";
import CreateReplicaSpec from './CreateReplicaSpec';

class CreateJob extends Component {
  constructor(props) {
    super(props);
    this.state = {
      name: "",
      namespace: "default",
      replicaSpecs: {},
      replicaCount: 0
    };

    this.handleInputChange = this.handleInputChange.bind(this);
    this.cancel = this.cancel.bind(this);
    this.deploy = this.deploy.bind(this);

    this.setReplicaSpec = this.setReplicaSpec.bind(this);
  }

  handleInputChange(event) {
    const target = event.target;
    const value = target.type === "checkbox" ? target.checked : target.value;
    const name = target.name;
    this.setState({
      [name]: value
    });
  }

  styles = {
    root: {
      display: "flex",
      flexDirection: "column"
    },
    divider: {
      marginTop: "20px",
      marginBottom: "5px"
    },
    field: {
      width: "80%"
    },
    addReplicaButton: {
      
    }
  };
  render() {
    return (
      <Card>
        <CardText style={this.styles.root}>
          <div>
            <RequiredTextField
              style={this.styles.field}
              floatingLabelText="Training name"
              name="name"
              onChange={this.handleInputChange}
            />
            <TextField
              style={this.styles.field}
              floatingLabelText="Namespace"
              name="namespace"
              value={this.state.namespace}
              onChange={this.handleInputChange}
            />

            {Object.keys(this.state.replicaSpecs).map(k => (
              <CreateReplicaSpec
                key={k}
                id={k}
                setReplicaSpec={this.setReplicaSpec}
              />
            ))}
            <Divider style={this.styles.divider} />
            <FlatButton
              style={this.styles.addReplicaButton}
              label="Add a replica type"
              primary={true}
              icon={<ContentAdd />}
              onClick={() => this.addReplicaSpec()}
            />
          </div>
        </CardText>
        <CardActions>
          <RaisedButton label="Deploy" primary={true} onClick={this.deploy} />
          <RaisedButton label="Cancel" onClick={this.cancel} />
        </CardActions>
      </Card>
    );
  }

  setReplicaSpec(id, spec) {
    this.setState({ replicaSpecs: { ...this.state.replicaSpecs, [id]: spec } });
  }

  addReplicaSpec() {
    const id = this.state.replicaCount;
    this.setState({
      replicaCount: id + 1,
      replicaSpecs: { ...this.state.replicaSpecs, [id]: {} }
    });
  }

  deploy() {
    // merge all the replicaSpecs into our final object
    let rs = {};
    for (let i in this.state.replicaSpecs) {
      rs = { ...rs, ...this.state.replicaSpecs[i] }
    }

    let spec = {
      metadata: {
        name: this.state.name,
        namespace: this.state.namespace
      },
      spec: {
        tfReplicaSpecs: rs
      }
    };

    createTFJobService(spec)
      .then(() => this.props.history.push("/"))
      .catch(console.error);
  }

  cancel() {
    this.props.history.goBack();
  }
}

CreateJob.propTypes = {
  history: PropTypes.object.isRequired
};

export default withRouter(CreateJob);
