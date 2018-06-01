import React, { Component } from "react";
import TextField from "material-ui/TextField";
import Divider from "material-ui/Divider";
import SelectField from "material-ui/SelectField";
import MenuItem from "material-ui/MenuItem";

import RequiredTextField from "./RequiredTextField";
import VolumeCreator from "./VolumeCreator";
import EnvVarCreator from "./EnvVarCreator";

const replicaTypes = {
  "Chief": 0,
  "Worker": 1,
  "PS": 2,
  "Eval": 3
};

class CreateReplicaSpec extends Component {
  constructor(props) {
    super(props);
    this.state = {
      image: "",
      command: "",
      args: "",
      gpus: 0,
      volumeSpec: {},
      envVars: [],
      replicaType: 1,
      replicas: 1
    };

    this.handleInputChange = this.handleInputChange.bind(this);
    this.setVolumesSpec = this.setVolumesSpec.bind(this);
    this.setEnvVars = this.setEnvVars.bind(this);
  }

  handleInputChange(event) {
    const target = event.target;
    const value = target.type === "checkbox" ? target.checked : target.value;
    const name = target.name;
    this.setState({
      [name]: value
    });
    this.bubbleSpec({ ...this.state, [name]: value });
  }

  styles = {
    divider: {
      marginTop: "20px",
      marginBottom: "5px"
    },
    field: {
      width: "80%"
    }
  };

  render() {
    return (
      <div>
        <Divider style={this.styles.divider} />
        <SelectField
          floatingLabelText="Replica Type"
          value={this.state.replicaType}
          onChange={(o, v) => {
            this.setState({ replicaType: v });
            this.bubbleSpec({ ...this.state, replicaType: v });
          }}>
          {Object.keys(replicaTypes).map((i, k) => (
            <MenuItem value={k} primaryText={i} key={i} />
          ))}
        </SelectField>
        <RequiredTextField
          style={this.styles.field}
          floatingLabelText="Container image"
          name="image"
          value={this.state.image}
          onChange={this.handleInputChange}
        />
        <TextField
          style={this.styles.field}
          floatingLabelText="Run command (comma separated)"
          name="command"
          value={this.state.command}
          onChange={this.handleInputChange}
        />
        <TextField
          style={this.styles.field}
          floatingLabelText="Run command arguments"
          name="args"
          value={this.state.args}
          onChange={this.handleInputChange}
        />
        <TextField
          style={this.styles.field}
          floatingLabelText="Replicas"
          type="number"
          min="0"
          name="replicas"
          value={this.state.replicas}
          onChange={this.handleInputChange}
        />
        <TextField
          style={this.styles.field}
          floatingLabelText="GPU(s) per replica"
          type="number"
          min="0"
          name="gpuCount"
          value={this.state.gpuCount}
          onChange={this.handleInputChange}
        />
        <EnvVarCreator setEnvVars={this.setEnvVars} />
        <VolumeCreator setVolumesSpec={this.setVolumesSpec} />
      </div>
    );
  }

  bubbleSpec(state) {
    this.props.setReplicaSpec(this.props.id, this.buildReplicaSpec(state));
  }

  buildReplicaSpec(state) {
    const args = state.args ? state.args.split(",").map(s => s.trim()) : [];
    const command = state.command ? state.command.split(",").map(s => s.trim()) : [];
    return {
      [Object.keys(replicaTypes)[state.replicaType]]: {
        replicas: parseInt(state.replicas, 10),
        template: {
          spec: {
            volumes: state.volumeSpec.volumes,
            containers: [
              {
                image: state.image,
                name: "tensorflow",
                command: command,
                args: args,
                env: state.envVars,
                volumeMounts: state.volumeSpec.volumeMounts
              }
            ],
            restartPolicy: "OnFailure"
          }
        }
      }
    };
  }

  setEnvVars(envVars) {
    this.setState({ envVars: envVars });
    this.bubbleSpec({ ...this.state, envVars: envVars });
  }

  setVolumesSpec(spec) {
    this.setState({ volumeSpec: spec });
    this.bubbleSpec({ ...this.state, volumeSpec: spec });
  }

}

export default CreateReplicaSpec;
