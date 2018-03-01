import React, { Component } from "react";
import PropTypes from "prop-types";
import { Card, CardText, CardActions } from "material-ui/Card";
import TextField from "material-ui/TextField";
import Divider from "material-ui/Divider";
import Toggle from "material-ui/Toggle";
import RaisedButton from "material-ui/RaisedButton";
import { withRouter } from "react-router-dom";

import { createTFJobService } from "../services";
import RequiredTextField from "./RequiredTextField";
import VolumeCreator from "./VolumeCreator";
import EnvVarCreator from "./EnvVarCreator";

class CreateJob extends Component {
  constructor(props) {
    super(props);
    this.state = {
      name: "",
      namespace: "default",
      masterImage: "",
      masterCommand: "",
      masterArgs: "",
      masterGpuCount: 0,
      workerImage: "",
      workerCommand: "",
      workerArgs: "",
      workerReplicas: 0,
      workerGpuCount: 0,
      psUseDefaultImage: true,
      psReplicas: 0,
      psImage: "",
      psCommand: "",
      psArgs: "",
      masterVolumeSpec: {},
      workerVolumeSpec: {},
      psVolumeSpec: {},
      masterEnvVars: [],
      workerEnvVars: [],
      psEnvVars: []
    };

    this.handleInputChange = this.handleInputChange.bind(this);
    this.cancel = this.cancel.bind(this);
    this.deploy = this.deploy.bind(this);
    this.setMasterVolumesSpec = this.setMasterVolumesSpec.bind(this);
    this.setPSVolumesSpec = this.setPSVolumesSpec.bind(this);
    this.setWorkerVolumesSpec = this.setWorkerVolumesSpec.bind(this);
    this.setMasterEnvVars = this.setMasterEnvVars.bind(this);
    this.setWorkerEnvVars = this.setWorkerEnvVars.bind(this);
    this.setPSEnvVars = this.setPSEnvVars.bind(this);
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
    divider: {
      marginTop: "20px",
      marginBottom: "5px"
    },
    header: {
      fontWeight: "bold",
      marginTop: "10px"
    },
    toggle: {
      width: "260px",
      marginTop: "10px"
    },
    root: {
      display: "flex",
      flexDirection: "column"
    },
    field: {
      width: "80%"
    }
  };
  render() {
    return (
      <Card>
        <CardText style={this.styles.root}>
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

          {/* MASTER */}
          <Divider style={this.styles.divider} />
          <h3 style={this.styles.header}>Master</h3>
          <RequiredTextField
            style={this.styles.field}
            floatingLabelText="Container image"
            name="masterImage"
            value={this.state.masterImage}
            onChange={this.handleInputChange}
          />
          <TextField
            style={this.styles.field}
            floatingLabelText="Run command (comma separated)"
            name="masterCommand"
            value={this.state.masterCommand}
            onChange={this.handleInputChange}
          />
          <TextField
            style={this.styles.field}
            floatingLabelText="Run command arguments"
            name="masterArgs"
            value={this.state.masterArgs}
            onChange={this.handleInputChange}
          />
          <TextField
            floatingLabelText="GPU(s) per replica"
            type="number"
            min="0"
            name="masterGpuCount"
            value={this.state.masterGpuCount}
            onChange={this.handleInputChange}
          />
          <EnvVarCreator setEnvVars={this.setMasterEnvVars} />
          <VolumeCreator setVolumesSpec={this.setMasterVolumesSpec} />

          {/* WORKER */}
          <Divider style={this.styles.divider} />
          <h3 style={this.styles.header}>Worker(s)</h3>
          <TextField
            style={this.styles.field}
            floatingLabelText="Container image"
            name="workerImage"
            value={this.state.workerImage}
            onChange={this.handleInputChange}
          />
          <TextField
            style={this.styles.field}
            floatingLabelText="Run command"
            name="workerCommand"
            value={this.state.workerCommand}
            onChange={this.handleInputChange}
          />
          <TextField
            style={this.styles.field}
            floatingLabelText="Run command arguments"
            name="workerArgs"
            value={this.state.workerArgs}
            onChange={this.handleInputChange}
          />
          <TextField
            floatingLabelText="Replicas"
            type="number"
            min="0"
            name="workerReplicas"
            value={this.state.workerReplicas}
            onChange={this.handleInputChange}
          />
          <TextField
            floatingLabelText="GPU(s) per replica"
            type="number"
            min="0"
            name="workerGpuCount"
            value={this.state.workerGpuCount}
            onChange={this.handleInputChange}
          />
          <EnvVarCreator setEnvVars={this.setWorkerEnvVars} />
          <VolumeCreator setVolumesSpec={this.setWorkerVolumesSpec} />

          {/* PARAMETER SERVER */}
          <Divider style={this.styles.divider} />
          <h3 style={this.styles.header}>Parameter Server(s)</h3>
          <TextField
            floatingLabelText="Replicas"
            name="psReplicas"
            type="number"
            min="0"
            value={this.state.psReplicas}
            onChange={this.handleInputChange}
          />
          <Toggle
            label="Use default image"
            defaultToggled={true}
            name="psUseDefaultImage"
            onToggle={this.handleInputChange}
            style={this.styles.toggle}
          />
          {!this.state.psUseDefaultImage && (
            <div style={{ display: "flex", flexDirection: "column" }}>
              <TextField
                style={this.styles.field}
                floatingLabelText="Container image"
                name="psImage"
                value={this.state.psImage}
                onChange={this.handleInputChange}
              />
              <TextField
                style={this.styles.field}
                floatingLabelText="Run command"
                name="psCommand"
                value={this.state.psCommand}
                onChange={this.handleInputChange}
              />
              <TextField
                style={this.styles.field}
                floatingLabelText="Run command arguments"
                name="psArgs"
                value={this.state.psArgs}
                onChange={this.handleInputChange}
              />
              <EnvVarCreator setEnvVars={this.setPSEnvVars} />
              <VolumeCreator setVolumesSpec={this.setPSVolumesSpec} />
            </div>
          )}
        </CardText>
        <CardActions>
          <RaisedButton label="Deploy" primary={true} onClick={this.deploy} />
          <RaisedButton label="Cancel" onClick={this.cancel} />
        </CardActions>
      </Card>
    );
  }

  deploy() {
    let rs = [
      this.newReplicaSpec(
        "MASTER",
        1,
        this.state.masterImage,
        this.state.masterCommand,
        this.state.masterArgs,
        this.state.masterEnvVars,
        this.state.masterVolumeSpec
      )
    ];
    if (this.state.workerReplicas > 0) {
      rs.push(
        this.newReplicaSpec(
          "WORKER",
          this.state.workerReplicas,
          this.state.workerImage,
          this.state.workerCommand,
          this.state.workerArgs,
          this.state.workerEnvVars,
          this.state.workerVolumeSpec
        )
      );
    }
    if (this.state.psReplicas > 0) {
      rs.push(
        this.newReplicaSpec(
          "PS",
          this.state.psReplicas,
          this.state.psImage,
          this.state.psCommand,
          this.state.psArgs,
          this.state.psEnvVars,
          this.state.psVolumeSpec
        )
      );
    }

    let spec = {
      metadata: {
        name: this.state.name,
        namespace: this.state.namespace
      },
      spec: {
        replicaSpecs: rs
      }
    };

    createTFJobService(spec)
      .then(() => this.props.history.push("/"))
      .catch(console.error);
  }

  cancel() {
    this.props.history.goBack();
  }

  newReplicaSpec(
    tfReplicaType,
    replicas,
    image,
    commandArr,
    argsArr,
    envVars,
    volumeSpec
  ) {
    const args = argsArr ? argsArr.split(",").map(s => s.trim()) : [];
    const command = commandArr ? commandArr.split(",").map(s => s.trim()) : [];
    return {
      replicas: parseInt(replicas, 10),
      tfReplicaType,
      template: {
        spec: {
          volumes: volumeSpec.volumes,
          containers: [
            {
              image,
              name: "tensorflow",
              command: command,
              args: args,
              env: envVars,
              volumeMounts: volumeSpec.volumeMounts
            }
          ],
          restartPolicy: "OnFailure"
        }
      }
    };
  }

  setMasterEnvVars(envVars) {
    this.setState({ masterEnvVars: envVars });
  }

  setWorkerEnvVars(envVars) {
    this.setState({ workerEnvVars: envVars });
  }

  setPSEnvVars(envVars) {
    this.setState({ psEnvVars: envVars });
  }

  setMasterVolumesSpec(spec) {
    this.setState({ masterVolumeSpec: spec });
  }

  setWorkerVolumesSpec(spec) {
    this.setState({ workerVolumeSpec: spec });
  }

  setPSVolumesSpec(spec) {
    this.setState({ psVolumeSpec: spec });
  }
}

CreateJob.propTypes = {
  history: PropTypes.object.isRequired
};

export default withRouter(CreateJob);
