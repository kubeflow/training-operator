import React, { Component } from "react";
import TextField from "material-ui/TextField";
import SelectField from "material-ui/SelectField";
import MenuItem from "material-ui/MenuItem";
import VolumeCreator from "./VolumeCreator";

class CreateTensorBoard extends Component {
  constructor(props) {
    super(props);
    this.state = {
      serviceType: 0,
      logDir: "/tmp/tensorflow",
      volumes: [],
      volumeMounts: []
    };
    this.handleInputChange = this.handleInputChange.bind(this);
    this.setVolumesSpec = this.setVolumesSpec.bind(this);
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
      <div style={this.styles.root}>
        <SelectField
          style={this.styles.field}
          floatingLabelText="Service"
          value={this.state.serviceType}
          onChange={(o, v) => {
            this.setState({ serviceType: v });
            this.bubbleSpec({ ...this.state, serviceType: v });
          }}
        >
          <MenuItem value={0} primaryText="Internal" />
          <MenuItem value={1} primaryText="External" />
        </SelectField>
        <TextField
          style={this.styles.field}
          floatingLabelText="Log dir"
          name="logDir"
          value={this.state.logDir}
          onChange={this.handleInputChange}
        />
        <VolumeCreator setVolumesSpec={this.setVolumesSpec} />
      </div>
    );
  }

  setVolumesSpec(volumeSpecs) {
    this.setState(Object.assign(this.state, volumeSpecs));
    this.bubbleSpec(Object.assign(this.state, volumeSpecs));
  }

  bubbleSpec(state) {
    this.props.setTensorBoardSpec(this.buildTensorBoardSpec(state));
  }

  buildTensorBoardSpec(state) {
    let st = state.serviceType === 0 ? "ClusterIP" : "LoadBalancer";
    return { ...state, serviceType: st };
  }
}

export default CreateTensorBoard;
